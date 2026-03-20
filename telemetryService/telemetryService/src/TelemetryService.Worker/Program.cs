using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using TelemetryService.Infrastructure.Messaging;
using TelemetryService.Infrastructure.Persistence;
using TelemetryService.Infrastructure.Configuration;

namespace TelemetryService;

internal class Program
{
    private static async Task Main(string[] args)
    {
        try
        {
            Console.WriteLine("🚀 Telemetry Service Starting...");


            LoadEnvironmentVariables();

            using var host = CreateHostBuilder(args).Build();

            var subscriber = host.Services.GetRequiredService<Subscriber>();
            var questDbSchemaManager = host.Services.GetRequiredService<QuestDbSchemaManager>();

            await questDbSchemaManager.EnsureOptimizedSchemaExists();

            Console.WriteLine("Starting telemetry service subscriber...");
            await subscriber.SubscribeAsync();

            await host.RunAsync();
        }
        catch (OutOfMemoryException ex)
        {
            Console.WriteLine($"OutOfMemoryException during service startup");
            Console.WriteLine($"   Message: {ex.Message}");
            Console.WriteLine($"   Stack Trace: {ex.StackTrace}");

            var process = System.Diagnostics.Process.GetCurrentProcess();
            var memoryUsageGB = (double)process.WorkingSet64 / (1024 * 1024 * 1024);
            Console.WriteLine($"   Memory Usage at Failure: {memoryUsageGB:F2}GB");

            Environment.Exit(1);
        }
        catch (Exception ex)
        {
            Console.WriteLine($" FATAL: Unhandled exception in telemetry service");
            Console.WriteLine($"   Exception Type: {ex.GetType().Name}");
            Console.WriteLine($"   Message: {ex.Message}");
            Console.WriteLine($"   Stack Trace: {ex.StackTrace}");

            if (ex.InnerException != null)
            {
                Console.WriteLine($"   Inner Exception: {ex.InnerException.GetType().Name}: {ex.InnerException.Message}");
            }

            Environment.Exit(1);
        }
    }

    private static void LoadEnvironmentVariables()
    {
        var root = Directory.GetCurrentDirectory();
        var possibleFiles = new[]
        {
            Path.Combine(root, ".env")
        };

        var anyFileLoaded = false;

        foreach (var file in possibleFiles)
            if (File.Exists(file))
            {
                Console.WriteLine($"Loading environment from {file}");
                DotEnv.Load(file);
                anyFileLoaded = true;
            }

        if (!anyFileLoaded) Console.WriteLine("No environment files found. Using container environment variables.");
    }

    private static IHostBuilder CreateHostBuilder(string[] args)
    {
        return Host.CreateDefaultBuilder(args)
            .ConfigureServices((_, services) =>
            {
                services.AddSingleton<Subscriber>();
                services.AddSingleton<QuestDbSchemaManager>(sp =>
                {
                    var host = Environment.GetEnvironmentVariable("QUESTDB_HTTP_HOST") ?? "questdb";
                    var port = Environment.GetEnvironmentVariable("QUESTDB_HTTP_PORT") ?? "9000";
                    var questDbUrl = $"{host}:{port}";
                    Console.WriteLine($"🔧 Initializing QuestDbSchemaManager with HTTP URL: http://{questDbUrl}");
                    return new QuestDbSchemaManager(questDbUrl);
                });
            });
    }
}