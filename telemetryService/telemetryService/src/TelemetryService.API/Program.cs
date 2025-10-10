using TelemetryService.Infrastructure.Configuration;
using TelemetryService.Infrastructure.Persistence;
using TelemetryService.Infrastructure.Messaging;
using TelemetryService.API.BackgroundServices; // Reference to your BackgroundService
using Prometheus;

var builder = WebApplication.CreateBuilder(args);

LoadEnvironmentVariables();

// Add API services
builder.Services.AddControllers();
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen(c =>
{
    c.SwaggerDoc("v1", new() { Title = "Telemetry Service API", Version = "v1" });
});

builder.Services.AddCors(options =>
{
    options.AddDefaultPolicy(policy =>
    {
        policy.AllowAnyOrigin()
              .AllowAnyMethod()
              .AllowAnyHeader();
    });
});

// Register telemetry services
builder.Services.AddSingleton<Subscriber>();

builder.Services.AddHostedService<TelemetryBackgroundService>();

builder.Logging.ClearProviders();
builder.Logging.AddConsole();

var app = builder.Build();

// Configure HTTP request pipeline
if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI(c =>
    {
        c.SwaggerEndpoint("/swagger/v1/swagger.json", "Telemetry Service API v1");
        c.RoutePrefix = string.Empty;
    });
}

app.UseCors();

// Enable Prometheus metrics
app.UseHttpMetrics();
app.UseRouting();

app.UseHttpsRedirection();
app.UseAuthorization();
app.MapControllers();
app.MapMetrics(); // Expose metrics at /metrics endpoint

Console.WriteLine("🚀 Telemetry Service starting...");
Console.WriteLine("✅ HTTP API endpoints enabled");
Console.WriteLine("✅ Background RabbitMQ processing enabled");
Console.WriteLine("✅ Prometheus metrics endpoint enabled at /metrics");
Console.WriteLine($"Environment: {app.Environment.EnvironmentName}");

app.Run();

static void LoadEnvironmentVariables()
{
    var root = Directory.GetCurrentDirectory();
    var envFile = Path.Combine(root, ".env");

    if (File.Exists(envFile))
    {
        Console.WriteLine($"Loading environment from {envFile}");
        DotEnv.Load(envFile);
    }
    else
    {
        Console.WriteLine("No .env file found. Using system environment variables.");
    }
}