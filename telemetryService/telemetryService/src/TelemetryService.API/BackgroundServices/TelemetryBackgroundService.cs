using TelemetryService.Infrastructure.Messaging;

namespace TelemetryService.API.BackgroundServices;

public class TelemetryBackgroundService : BackgroundService
{
    private readonly Subscriber _subscriber;
    private readonly ILogger<TelemetryBackgroundService> _logger;

    public TelemetryBackgroundService(Subscriber subscriber, ILogger<TelemetryBackgroundService> logger)
    {
        _subscriber = subscriber;
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        _logger.LogInformation("🔄 TelemetryBackgroundService starting...");

        try
        {
            await _subscriber.SubscribeAsync();
        }
        catch (OperationCanceledException)
        {
            _logger.LogInformation("🛑 TelemetryBackgroundService was cancelled");
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "❌ Fatal error in TelemetryBackgroundService");
            throw;
        }
    }

    public override async Task StopAsync(CancellationToken stoppingToken)
    {
        _logger.LogInformation("🛑 TelemetryBackgroundService stopping...");
        await base.StopAsync(stoppingToken);
    }
}