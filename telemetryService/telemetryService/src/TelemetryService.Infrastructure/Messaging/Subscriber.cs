using System.Text;
using RabbitMQ.Client;
using RabbitMQ.Client.Events;
using TelemetryService.Domain.Models;
using TelemetryService.Infrastructure.Persistence;

namespace TelemetryService.Infrastructure.Messaging;

public class Subscriber

{
    private const int MaxRetryAttempts = 10;
    private const int RetryDelayMs = 5000;
    private const int MaxConcurrentProcessing = 25;
    private const int BasePollIntervalMs = 10; // Base polling interval when messages available
    private const int EmptyQueueDelayMs = 100; // Delay when no messages available
    private const int BatchSize = 30; // Number of messages to pull in each batch
    
    private readonly QuestDbService _questDbService;
    private readonly SemaphoreSlim _processingSemaphore = new(MaxConcurrentProcessing, MaxConcurrentProcessing);
    private volatile bool _pauseProcessing = false;
    private volatile bool _stopRequested = false;
    private Timer? _memoryMonitorTimer;
    private double _lastLoggedMemoryUsage = 0;

    public Subscriber(QuestDbService questDbService)
    {
        _questDbService = questDbService;
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

        await channel.BasicQosAsync(0, 1000, false);

        await channel.QueueBindAsync(
            "telemetry_queue",
            "telemetry_topic",
            "telemetry.ticks");
        Console.WriteLine("Queue bound to exchange with routing key telemetry.ticks");

        Console.WriteLine("🔄 Starting pull-based message consumption...");
        
        // Start memory monitoring
        StartMemoryMonitoring();

        // Start pull-based consumption loop
        await StartPullBasedConsumption(channel);
    }

    private async Task StartPullBasedConsumption(IChannel channel)
    {
        var messagesProcessed = 0;
        var lastStatsTime = DateTime.UtcNow;
        
        Console.WriteLine("📥 Ready to pull messages from queue...");

        while (!_stopRequested)
        {
            try
            {
                // Skip processing if memory pressure is too high
                if (_pauseProcessing)
                {
                    await Task.Delay(EmptyQueueDelayMs);
                    continue;
                }

                // Pull a batch of messages
                var messagesPulled = 0;
                var pullTasks = new List<Task>();

                for (int i = 0; i < BatchSize && _processingSemaphore.CurrentCount > 0; i++)
                {
                    var result = await channel.BasicGetAsync("telemetry_queue", false);
                    
                    if (result != null)
                    {
                        messagesPulled++;
                        
                        // Process message in background task
                        var processTask = ProcessMessageAsync(channel, result);
                        pullTasks.Add(processTask);
                    }
                    else
                    {
                        // No more messages available
                        break;
                    }
                }

                // Log processing stats periodically
                messagesProcessed += messagesPulled;
                if (DateTime.UtcNow - lastStatsTime > TimeSpan.FromSeconds(10))
                {
                    var rate = messagesProcessed / (DateTime.UtcNow - lastStatsTime).TotalSeconds;
                    Console.WriteLine($"📊 Pull Stats: {messagesProcessed} msgs processed, {rate:F1} msgs/sec, {_processingSemaphore.CurrentCount}/{MaxConcurrentProcessing} threads available");
                    messagesProcessed = 0;
                    lastStatsTime = DateTime.UtcNow;
                }

                // Wait for processing tasks to complete or use appropriate delay
                if (pullTasks.Any())
                {
                    // Messages were pulled, short delay before next batch
                    await Task.Delay(BasePollIntervalMs);
                }
                else
                {
                    // No messages available, longer delay to avoid tight polling
                    await Task.Delay(EmptyQueueDelayMs);
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"❌ Error in pull-based consumption loop: {ex.GetType().Name}");
                Console.WriteLine($"   Message: {ex.Message}");
                Console.WriteLine($"   Stack Trace: {ex.StackTrace}");
                
                // Wait before retrying to avoid tight error loop
                await Task.Delay(RetryDelayMs);
            }
        }
    }

    private async Task ProcessMessageAsync(IChannel channel, BasicGetResult result)
    {
        await _processingSemaphore.WaitAsync();
        
        try
        {
            var body = result.Body.ToArray();
            var message = TelemetryBatch.Parser.ParseFrom(body);
            
            // Write to QuestDB synchronously but don't wait for completion
            // This provides natural backpressure without overwhelming QuestDB
            await _questDbService.WriteBatch(message);
            
            // Acknowledge message after successful QuestDB write
            await channel.BasicAckAsync(result.DeliveryTag, false);
        }
        catch (OutOfMemoryException ex)
        {
            Console.WriteLine($"❌ CRITICAL: OutOfMemoryException during message processing");
            Console.WriteLine($"   Message: {ex.Message}");
            Console.WriteLine($"   Stack Trace: {ex.StackTrace}");
            Console.WriteLine($"   DeliveryTag: {result.DeliveryTag}");
            
            // Log current memory state
            var process = System.Diagnostics.Process.GetCurrentProcess();
            var memoryUsageGB = (double)process.WorkingSet64 / (1024 * 1024 * 1024);
            var gcMemoryMB = GC.GetTotalMemory(false) / (1024 * 1024);
            
            Console.WriteLine($"   Current Memory Usage: {memoryUsageGB:F2}GB (Working Set)");
            Console.WriteLine($"   GC Memory: {gcMemoryMB:F2}MB");
            Console.WriteLine($"   Processing Semaphore Count: {_processingSemaphore.CurrentCount}/{MaxConcurrentProcessing}");
            
            // Force garbage collection
            Console.WriteLine("🧹 Forcing garbage collection...");
            GC.Collect();
            GC.WaitForPendingFinalizers();
            GC.Collect();
            
            var gcMemoryAfterMB = GC.GetTotalMemory(false) / (1024 * 1024);
            Console.WriteLine($"   GC Memory after cleanup: {gcMemoryAfterMB:F2}MB");
            
            // Reject message (requeue for retry)
            await channel.BasicNackAsync(result.DeliveryTag, false, true);
            
            // Pause processing temporarily and set up recovery
            _pauseProcessing = true;
            Console.WriteLine("⏸️  Processing PAUSED due to OutOfMemoryException");
            
            // Schedule recovery attempt
            _ = Task.Run(async () =>
            {
                await Task.Delay(TimeSpan.FromSeconds(30));
                
                var processAfterDelay = System.Diagnostics.Process.GetCurrentProcess();
                var memoryAfterDelay = (double)processAfterDelay.WorkingSet64 / (1024 * 1024 * 1024);
                
                if (memoryAfterDelay < 4.0) // If memory usage dropped below 4GB
                {
                    _pauseProcessing = false;
                    Console.WriteLine($"✅ Processing RESUMED after memory recovery (Memory: {memoryAfterDelay:F2}GB)");
                }
                else
                {
                    Console.WriteLine($"⚠️  Memory still high after delay (Memory: {memoryAfterDelay:F2}GB) - keeping processing paused");
                }
            });
        }
        catch (Exception ex)
        {
            Console.WriteLine($"❌ Error processing message: {ex.GetType().Name}");
            Console.WriteLine($"   Message: {ex.Message}");
            Console.WriteLine($"   Stack Trace: {ex.StackTrace}");
            Console.WriteLine($"   DeliveryTag: {result.DeliveryTag}");
            
            // Log additional context for debugging
            var process = System.Diagnostics.Process.GetCurrentProcess();
            var memoryUsageGB = (double)process.WorkingSet64 / (1024 * 1024 * 1024);
            Console.WriteLine($"   Current Memory Usage: {memoryUsageGB:F2}GB");
            Console.WriteLine($"   Processing Semaphore Count: {_processingSemaphore.CurrentCount}/{MaxConcurrentProcessing}");
            
            if (ex.InnerException != null)
            {
                Console.WriteLine($"   Inner Exception: {ex.InnerException.GetType().Name}: {ex.InnerException.Message}");
            }
            
            // Reject message (requeue for retry)
            await channel.BasicNackAsync(result.DeliveryTag, false, true);
        }
        finally
        {
            _processingSemaphore.Release();
        }
    }
    
    private void StartMemoryMonitoring()
    {
        _memoryMonitorTimer = new Timer(CheckMemoryPressure, null, TimeSpan.Zero, TimeSpan.FromSeconds(5));
    }
    
    private void CheckMemoryPressure(object? state)
    {
        try
        {
            var currentMemory = GC.GetTotalMemory(false);
            var process = System.Diagnostics.Process.GetCurrentProcess();
            var memoryUsagePercent = (double)process.WorkingSet64 / (1024 * 1024 * 1024); // Convert to GB
            var gcMemoryMB = currentMemory / (1024 * 1024);
            
            // Simple heuristic: if working set > 5GB (83% of 6GB limit), pause processing
            var shouldPause = memoryUsagePercent > 5.0;
            
            // Log memory changes when significant (>0.5GB change) or status changes
            var memoryDelta = Math.Abs(memoryUsagePercent - _lastLoggedMemoryUsage);
            var statusChanged = shouldPause != _pauseProcessing;
            
            if (statusChanged || memoryDelta > 0.5)
            {
                Console.WriteLine($"📊 Memory Status: {memoryUsagePercent:F2}GB (Working Set), {gcMemoryMB:F0}MB (GC), Processing: {(_pauseProcessing ? "PAUSED" : "ACTIVE")}, Semaphore: {_processingSemaphore.CurrentCount}/{MaxConcurrentProcessing}");
                _lastLoggedMemoryUsage = memoryUsagePercent;
            }
            
            if (statusChanged)
            {
                _pauseProcessing = shouldPause;
                Console.WriteLine($"🔄 Memory pressure {(shouldPause ? "HIGH" : "NORMAL")} - Consumer {(shouldPause ? "PAUSED" : "RESUMED")} (Memory: {memoryUsagePercent:F1}GB)");
            }
            
            // Warning if memory is approaching critical levels
            if (memoryUsagePercent > 4.5 && !shouldPause)
            {
                Console.WriteLine($"⚠️  Memory approaching critical level: {memoryUsagePercent:F2}GB");
            }
        }
        catch (Exception ex)
        {
            Console.WriteLine($"❌ Error monitoring memory: {ex.GetType().Name}: {ex.Message}");
        }
    }
    
    public void Dispose()
    {
        _stopRequested = true;
        _memoryMonitorTimer?.Dispose();
        _processingSemaphore?.Dispose();
    }
}