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
    private const int MaxConcurrentProcessing = 50;
    private const double MemoryThresholdPercent = 0.80;
    private readonly QuestDbService _questDbService;
    private readonly SemaphoreSlim _processingSemaphore = new(MaxConcurrentProcessing, MaxConcurrentProcessing);
    private volatile bool _pauseProcessing = false;
    private Timer? _memoryMonitorTimer;

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
        await channel.BasicQosAsync(0, 200, false);
        Console.WriteLine("Channel created successfully");

        await channel.ExchangeDeclareAsync(
            "telemetry_topic",
            ExchangeType.Topic,
            true,
            false);
        Console.WriteLine("Exchange declared successfully");

        await channel.QueueDeclareAsync(
            "telemetry_queue",
            true,
            false,
            false);
        Console.WriteLine("Queue declared successfully");

        await channel.QueueBindAsync(
            "telemetry_queue",
            "telemetry_topic",
            "telemetry.ticks");
        Console.WriteLine("Queue bound to exchange with routing key telemetry.ticks");

        Console.WriteLine("Waiting for messages...");
        
        // Start memory monitoring
        StartMemoryMonitoring();

        var consumer = new AsyncEventingBasicConsumer(channel);
        consumer.ReceivedAsync += async (_, ea) =>
        {
            // Skip processing if memory pressure is too high
            if (_pauseProcessing)
            {
                await Task.Delay(100); // Brief delay to avoid tight loop
                return;
            }

            // Limit concurrent processing
            await _processingSemaphore.WaitAsync();
            try
            {
                var body = ea.Body.ToArray();
                var message = TelemetryBatch.Parser.ParseFrom(body);
                
                var questTask = _questDbService.WriteBatch(message);
                
                await Task.WhenAll(questTask);
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error processing message: {ex.Message}");
            }
            finally
            {
                _processingSemaphore.Release();
            }
        };

        await channel.BasicConsumeAsync(
            "telemetry_queue",
            true,
            consumer);

        Console.WriteLine("Consumer registered. Waiting for messages...");

        await Task.Delay(Timeout.Infinite);
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
            
            // Simple heuristic: if working set > 5GB (83% of 6GB limit), pause processing
            var shouldPause = memoryUsagePercent > 5.0;
            
            if (shouldPause != _pauseProcessing)
            {
                _pauseProcessing = shouldPause;
                Console.WriteLine($"🔄 Memory pressure {(shouldPause ? "HIGH" : "NORMAL")} - Consumer {(shouldPause ? "PAUSED" : "RESUMED")} (Memory: {memoryUsagePercent:F1}GB)");
            }
        }
        catch (Exception ex)
        {
            Console.WriteLine($"Error monitoring memory: {ex.Message}");
        }
    }
    
    public void Dispose()
    {
        _memoryMonitorTimer?.Dispose();
        _processingSemaphore?.Dispose();
    }
}