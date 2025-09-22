using QuestDB;
using QuestDB.Senders;
using TelemetryService.Domain.Models;

namespace TelemetryService.Infrastructure.Persistence;

public class QuestDbService : IDisposable
{
    private readonly ISender _sender;
    private readonly string _tableName;
    private bool _disposed;

    public QuestDbService()
    {
        var host = Environment.GetEnvironmentVariable("QUESTDB_TCP_HOST") ?? "questdb";
        var port = int.TryParse(Environment.GetEnvironmentVariable("QUESTDB_TCP_PORT"), out var p) ? p : 9009;
        _tableName = Environment.GetEnvironmentVariable("QUESTDB_TABLE_NAME") ?? "TelemetryTicks";

        // Try different connection string formats for QuestDB community edition
        var connectionStrings = new[]
        {
            // Standard TCP format with explicit settings
            $"tcp::addr={host}:{port};auto_flush_rows=100;auto_flush_interval=1000;",
            // Minimal TCP format
            $"tcp::addr={host}:{port};",
            // TCP with protocol version
            $"tcp::addr={host}:{port};protocol_version=1;",
            // HTTP fallback for debugging
            $"http::addr={host}:9000;auto_flush_rows=100;auto_flush_interval=1000;"
        };

        Exception? lastException = null;

        foreach (var connectionString in connectionStrings)
        {
            try
            {
                Console.WriteLine($"🔗 Attempting QuestDB connection: {connectionString}");
                _sender = Sender.New(connectionString);
                Console.WriteLine($"✅ Successfully created QuestDB connection: {connectionString} → {_tableName}");
                return; // Success - exit constructor
            }
            catch (Exception ex)
            {
                Console.WriteLine($"❌ Failed connection attempt: {ex.Message}");
                lastException = ex;
            }
        }

        // If all connection attempts failed, throw the last exception
        Console.WriteLine($"❌ All QuestDB connection attempts failed");
        throw new InvalidOperationException($"Failed to connect to QuestDB after trying multiple connection strings", lastException);
    }

    public async Task WriteBatch(TelemetryBatch batch)
    {
        if (_disposed)
            throw new ObjectDisposedException(nameof(QuestDbService));

        if (batch?.Records == null || !batch.Records.Any())
        {
            Console.WriteLine("⚠️  Empty batch received, skipping");
            return;
        }

        var validRecords = batch.Records.Where(IsValidRecord).ToList();
        if (!validRecords.Any())
        {
            Console.WriteLine("⚠️  No valid records in batch, skipping");
            return;
        }

        const int maxRetries = 3;
        var retryCount = 0;

        while (retryCount <= maxRetries)
        {
            try
            {
                await WriteRecordsInternal(validRecords, batch.BatchId);
                return; // Success - exit retry loop
            }
            catch (Exception ex) when (IsRetryableError(ex) && retryCount < maxRetries)
            {
                retryCount++;
                var delay = Math.Min(1000 * retryCount, 5000); // Cap at 5 seconds
                Console.WriteLine($"⚠️  Attempt {retryCount} failed, retrying in {delay}ms: {ex.Message}");
                await Task.Delay(delay);
            }
        }

        Console.WriteLine($"❌ Failed to write batch {batch.BatchId} after {maxRetries + 1} attempts");
        throw new InvalidOperationException($"Failed to write batch after {maxRetries + 1} attempts");
    }

    private async Task WriteRecordsInternal(List<Telemetry> validRecords, string batchId)
    {
        var processedCount = 0;

        foreach (var record in validRecords)
        {
            await _sender.Table(_tableName)
                        .Symbol("session_id", Sanitize(record.SessionId))
                        .Symbol("track_name", Sanitize(record.TrackName))
                        .Symbol("track_id", Sanitize(record.TrackId))
                        .Symbol("lap_id", Sanitize(record.LapId))
                        .Symbol("session_num", Sanitize(record.SessionNum))
                        .Symbol("session_type", Sanitize(record.SessionType))
                        .Symbol("session_name", Sanitize(record.SessionName))

                        .Column("car_id", Sanitize(record.CarId))
                        .Column("gear", record.Gear)
                        .Column("player_car_position", Math.Max(0, (long)record.PlayerCarPosition))

                        .Column("speed", ValidateDouble(record.Speed))
                        .Column("lap_dist_pct", ValidateDouble(record.LapDistPct))
                        .Column("session_time", ValidateDouble(record.SessionTime))
                        .Column("lat", ValidateDouble(record.Lat))
                        .Column("lon", ValidateDouble(record.Lon))
                        .Column("lap_current_lap_time", ValidateDouble(record.LapCurrentLapTime))
                        .Column("lapLastLapTime", ValidateDouble(record.LapLastLapTime))
                        .Column("lapDeltaToBestLap", ValidateDouble(record.LapDeltaToBestLap))

                        .Column("throttle", (float)ValidateDouble(record.Throttle))
                        .Column("brake", (float)ValidateDouble(record.Brake))
                        .Column("steering_wheel_angle", (float)ValidateDouble(record.SteeringWheelAngle))
                        .Column("rpm", (float)ValidateDouble(record.Rpm))
                        .Column("velocity_x", (float)ValidateDouble(record.VelocityX))
                        .Column("velocity_y", (float)ValidateDouble(record.VelocityY))
                        .Column("velocity_z", (float)ValidateDouble(record.VelocityZ))
                        .Column("fuel_level", (float)ValidateDouble(record.FuelLevel))
                        .Column("alt", (float)ValidateDouble(record.Alt))
                        .Column("lat_accel", (float)ValidateDouble(record.LatAccel))
                        .Column("long_accel", (float)ValidateDouble(record.LongAccel))
                        .Column("vert_accel", (float)ValidateDouble(record.VertAccel))
                        .Column("pitch", (float)ValidateDouble(record.Pitch))
                        .Column("roll", (float)ValidateDouble(record.Roll))
                        .Column("yaw", (float)ValidateDouble(record.Yaw))
                        .Column("yaw_north", (float)ValidateDouble(record.YawNorth))
                        .Column("voltage", (float)ValidateDouble(record.Voltage))
                        .Column("waterTemp", (float)ValidateDouble(record.WaterTemp))
                        .Column("lFpressure", (float)ValidateDouble(record.LFpressure))
                        .Column("rFpressure", (float)ValidateDouble(record.RFpressure))
                        .Column("lRpressure", (float)ValidateDouble(record.LRpressure))
                        .Column("rRpressure", (float)ValidateDouble(record.RRpressure))
                        .Column("lFtempM", (float)ValidateDouble(record.LFtempM))
                        .Column("rFtempM", (float)ValidateDouble(record.RFtempM))
                        .Column("lRtempM", (float)ValidateDouble(record.LRtempM))
                        .Column("rRtempM", (float)ValidateDouble(record.RRtempM))
                        .AtAsync(record.TickTime.ToDateTime());

            processedCount++;
            if (processedCount % 1000 == 0)
            {
                Console.WriteLine($"   📊 {processedCount}/{validRecords.Count} records processed");
            }
        }

        Console.WriteLine($"✅ Successfully wrote {processedCount} records to QuestDB (Batch: {batchId})");
    }

    private static bool IsRetryableError(Exception ex)
    {
        var message = ex.Message?.ToLower() ?? "";
        var innerMessage = ex.InnerException?.Message?.ToLower() ?? "";

        return ex is IOException ||
               message.Contains("socket") ||
               message.Contains("connection reset") ||
               message.Contains("could not write data") ||
               message.Contains("transport connection") ||
               innerMessage.Contains("connection reset") ||
               innerMessage.Contains("transport connection");
    }

    private static bool IsValidRecord(Telemetry record)
    {
        return !string.IsNullOrWhiteSpace(record.SessionId) ||
               !string.IsNullOrWhiteSpace(record.TrackName);
    }

    private static string Sanitize(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
            return "unknown";

        return value.Replace(",", "_")
                   .Replace(" ", "_")
                   .Replace("=", "_")
                   .Replace("\n", "_")
                   .Replace("\r", "_")
                   .Replace("\"", "_")
                   .Replace("'", "_")
                   .Replace("\\", "_")
                   .Trim();
    }

    private static double ValidateDouble(double value)
    {
        return double.IsNaN(value) || double.IsInfinity(value) ||
               value == double.MinValue || value == double.MaxValue ? 0.0 : value;
    }

    public async Task<bool> TriggerSchemaOptimization()
    {
        // Schema optimization not needed with per-batch pattern
        Console.WriteLine("🔧 Schema optimization not applicable for per-batch connections");
        return await Task.FromResult(true);
    }

    public void Dispose()
    {
        if (_disposed) return;

        try
        {
            _sender?.Dispose();
        }
        catch (Exception ex)
        {
            Console.WriteLine($"⚠️  Error disposing QuestDB sender: {ex.Message}");
        }

        _disposed = true;
    }
}