using System.Buffers;
using System.Threading.Channels;
using RabbitMQ.Client;
using RabbitMQ.Client.Events;
using TelemetryService.Domain.Models;
using TelemetryService.Infrastructure.Persistence;

namespace TelemetryService.Infrastructure.Messaging;

public class Subscriber

{
    private const int MaxRetryAttempts = 10;
    private const int RetryDelayMs = 5000;

    private volatile bool _stopRequested = false;

    private readonly Channel<(TelemetryBatch batch, ulong deliveryTag, IChannel channel)> _messageChannel = Channel.CreateUnbounded<(TelemetryBatch, ulong, IChannel)>();

    public Subscriber()
    {
    }

    public async Task SubscribeAsync()
    {
        var retryCount = 0;
        var connected = false;

        while (!connected && retryCount < MaxRetryAttempts)
            try
            {
                Console.WriteLine($"Connecting to RabbitMQ (Attempt {retryCount + 1}/{MaxRetryAttempts})...");
                await ConnectAndSubscribeAsync();
                connected = true;
            }
            catch (Exception ex)
            {
                retryCount++;
                Console.WriteLine($"Failed to connect to RabbitMQ: {ex.Message}");
                Console.WriteLine($"Exception details: {ex}");
                Console.WriteLine($"Retrying in {RetryDelayMs / 1000} seconds...");
                await Task.Delay(RetryDelayMs);
            }

        if (!connected) Console.WriteLine("Failed to connect to RabbitMQ after multiple attempts. Exiting.");
    }

    private async Task ConnectAndSubscribeAsync()
    {
        var factory = new ConnectionFactory();
        factory.Uri = new Uri("amqp://admin:changeme@rabbitmq:5672/");
        factory.RequestedHeartbeat = TimeSpan.FromSeconds(60);
        factory.AutomaticRecoveryEnabled = true;
        factory.NetworkRecoveryInterval = TimeSpan.FromSeconds(10);

        Console.WriteLine($"Connecting to RabbitMQ at {factory.HostName}:{factory.Port} with user {factory.UserName}");

        using var connection = await factory.CreateConnectionAsync();
        Console.WriteLine("Successfully connected to RabbitMQ!");

        using var channel = await connection.CreateChannelAsync();

        await channel.BasicQosAsync(0, 5000, false);

        await channel.QueueBindAsync(
            "telemetry_queue",
            "telemetry_topic",
            "telemetry.ticks");

        Console.WriteLine("Queue bound to exchange with routing key telemetry.ticks");

        Console.WriteLine("🔄 Starting push-based message consumption with batching...");


        await StartPushBasedConsumption(channel);
    }

    private async Task StartPushBasedConsumption(IChannel channel)
    {
        var consumer = new AsyncEventingBasicConsumer(channel);
        var messagesProcessed = 0;
        var lastStatsTime = DateTime.UtcNow;

        Console.WriteLine("📥 Ready to consume messages from queue...");

        consumer.ReceivedAsync += async (sender, ea) =>
        {
            if (_stopRequested) return;


            var bodyLength = ea.Body.Length;
            var buffer = ArrayPool<byte>.Shared.Rent(bodyLength);
            try
            {
                ea.Body.CopyTo(buffer);
                var message = TelemetryBatch.Parser.ParseFrom(buffer, 0, bodyLength);

                await _messageChannel.Writer.WriteAsync((message, ea.DeliveryTag, channel));

                Interlocked.Increment(ref messagesProcessed);
            }
            finally
            {
                ArrayPool<byte>.Shared.Return(buffer);
            }
        };

        await channel.BasicConsumeAsync(queue: "telemetry_queue", autoAck: false, consumer: consumer);

        _ = Task.Run(() => ProcessBatchMessages());
        while (!_stopRequested)
        {
            await Task.Delay(5000); // Log every 5 seconds
            messagesProcessed = 0;
            lastStatsTime = DateTime.UtcNow;
        }
    }

    private async Task ProcessBatchMessages()
    {
        const int targetBatchSize = 20;
        const int maxRecordsPerBatch = 25000;
        const int batchTimeoutMs = 5000;

        var batchBuffer = new List<(TelemetryBatch batch, ulong deliveryTag, IChannel channel)>(targetBatchSize);
        (TelemetryBatch batch, ulong deliveryTag, IChannel channel)? pendingItem = null;

        Console.WriteLine($"📦 Batch processor started (target: {targetBatchSize} msgs, max: {maxRecordsPerBatch} records, timeout: {batchTimeoutMs}ms)");

        try
        {
            while (!_stopRequested)
            {
                var deadline = DateTime.UtcNow.AddMilliseconds(batchTimeoutMs);

                // Add pending item from previous batch if exists
                if (pendingItem.HasValue)
                {
                    batchBuffer.Add(pendingItem.Value);
                    pendingItem = null;
                }

                while (batchBuffer.Count < targetBatchSize && DateTime.UtcNow < deadline)
                {
                    var remaining = (deadline - DateTime.UtcNow).TotalMilliseconds;
                    if (remaining <= 0) break;

                    var cts = new CancellationTokenSource(TimeSpan.FromMilliseconds(remaining));
                    try
                    {
                        if (await _messageChannel.Reader.WaitToReadAsync(cts.Token))
                        {
                            while (_messageChannel.Reader.TryRead(out var item))
                            {
                                // Check if adding this item would exceed record limit
                                var currentRecordCount = batchBuffer.Sum(b => b.batch.Records.Count);
                                var potentialRecordCount = currentRecordCount + item.batch.Records.Count;

                                if (potentialRecordCount > maxRecordsPerBatch && batchBuffer.Count > 0)
                                {
                                    // Would exceed limit - save for next batch
                                    Console.WriteLine($"   Record limit would be exceeded: {currentRecordCount} + {item.batch.Records.Count} > {maxRecordsPerBatch}");
                                    pendingItem = item;
                                    break;
                                }

                                batchBuffer.Add(item);

                                if (batchBuffer.Count >= targetBatchSize)
                                {
                                    break;
                                }
                            }
                        }
                    }
                    catch (OperationCanceledException)
                    {
                        break;
                    }

                    // Break if we have a pending item (batch is full)
                    if (pendingItem.HasValue)
                    {
                        break;
                    }
                }


                if (batchBuffer.Count > 0)
                {
                    try
                    {
                        var allRecords = batchBuffer
                        .SelectMany(b => b.batch.Records)
                        .Where(QuestDbService.IsValidRecord)
                        .ToList();

                        if (allRecords.Any())
                        {
                            try
                            {
                                await QuestDbService.WriteBatch(allRecords);

                                Console.WriteLine($"📦 Flushed {batchBuffer.Count} messages ({allRecords.Count} records)");

                                // Try to ACK with timeout protection
                                try
                                {
                                    using var ackCts = new CancellationTokenSource(TimeSpan.FromSeconds(5));
                                    foreach (var item in batchBuffer)
                                    {
                                        await item.channel.BasicAckAsync(item.deliveryTag, false, ackCts.Token);
                                    }
                                }
                                catch (OperationCanceledException)
                                {
                                    Console.WriteLine($"⚠️  ACK timeout - channel may be slow");
                                }
                                catch (Exception ackEx)
                                {
                                    Console.WriteLine($"⚠️  Could not ACK messages (channel closed): {ackEx.Message}");
                                    Console.WriteLine($"   Messages will be re-delivered by RabbitMQ");
                                }
                            }
                            catch (Exception ex)
                            {
                                Console.WriteLine($"❌ Batch write failed: {ex.Message}");

                                // Try to NACK with timeout protection
                                try
                                {
                                    using var nackCts = new CancellationTokenSource(TimeSpan.FromSeconds(5));
                                    foreach (var item in batchBuffer)
                                    {
                                        await item.channel.BasicNackAsync(item.deliveryTag, false, true, nackCts.Token);
                                    }
                                }
                                catch (OperationCanceledException)
                                {
                                    Console.WriteLine($"⚠️  NACK timeout - channel may be slow");
                                }
                                catch (Exception nackEx)
                                {
                                    Console.WriteLine($"⚠️  Could not NACK messages (channel closed): {nackEx.Message}");
                                    Console.WriteLine($"   Messages will be re-delivered by RabbitMQ");
                                }
                            }
                        }

                    }
                    finally
                    {
                        batchBuffer.Clear();
                    }
                }
            }
        }
        finally
        {
            Console.WriteLine("📦 Batch processor stopped");
        }
    }

    public void Dispose()
    {
        _stopRequested = true;
    }
}