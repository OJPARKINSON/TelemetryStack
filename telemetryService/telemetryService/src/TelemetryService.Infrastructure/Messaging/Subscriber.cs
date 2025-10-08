using System.Buffers;
using System.Threading.Channels;
using RabbitMQ.Client;
using RabbitMQ.Client.Events;
using TelemetryService.Domain.Models;
using TelemetryService.Infrastructure.Persistance;
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
        const int targetBatchSize = 2000;
        const int batchTimeoutMs = 500;

        var batchBuffer = new List<(TelemetryBatch batch, ulong deliveryTag, IChannel channel)>(targetBatchSize);
        var questSender = QuestDbSenderPool.Get();

        Console.WriteLine($"📦 Batch processor started (target: {targetBatchSize} msgs, timeout: {batchTimeoutMs}ms)");

        try
        {
            while (!_stopRequested)
            {
                var deadline = DateTime.UtcNow.AddMilliseconds(batchTimeoutMs);

                while (batchBuffer.Count < targetBatchSize && DateTime.UtcNow < deadline)
                {
                    var remaining = (deadline - DateTime.UtcNow).TotalMilliseconds;
                    if (remaining <= 0) break;

                    var cts = new CancellationTokenSource(TimeSpan.FromMilliseconds(remaining));
                    try
                    {
                        if (await _messageChannel.Reader.WaitToReadAsync(cts.Token))
                        {
                            while (_messageChannel.Reader.TryRead(out var item) && batchBuffer.Count < targetBatchSize)
                            {
                                batchBuffer.Add(item);
                            }
                        }
                    }
                    catch (OperationCanceledException)
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
                                await QuestDbService.WriteBatch(questSender, allRecords);

                            }
                            catch (Exception ex)
                            {
                                foreach (var item in batchBuffer)
                                {
                                    await item.channel.BasicNackAsync(item.deliveryTag, false, true);
                                }
                            }

                            Console.WriteLine($"📦 Flushed {batchBuffer.Count} messages ({allRecords.Count} records)");
                        }

                        foreach (var item in batchBuffer)
                        {
                            await item.channel.BasicAckAsync(item.deliveryTag, false);
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
            QuestDbSenderPool.Return(questSender);
            Console.WriteLine("📦 Batch processor stopped");
        }
    }

    public void Dispose()
    {
        _stopRequested = true;
    }
}