using QuestDB;
using QuestDB.Senders;
using TelemetryService.Domain.Models;

namespace TelemetryService.Infrastructure.Persistence;

public class QuestDbService
{
    private ISender? _sender;
    private QuestDbSchemaManager? _schemaManager;
    
    // Circuit breaker properties
    private int _consecutiveErrors = 0;
    private DateTime _lastErrorTime = DateTime.MinValue;
    private const int MaxConsecutiveErrors = 5;
    private const int CircuitBreakerTimeoutMinutes = 2;
    private bool _circuitBreakerOpen = false;

    public QuestDbService()
    {
        string? url = Environment.GetEnvironmentVariable("QUESTDB_URL");

        if (url == null)
        {
            return;
        }

        _schemaManager = new QuestDbSchemaManager(url);

        _sender = Sender.New($"http::addr={url.Replace("http://", "")};auto_flush_rows=2000;auto_flush_interval=500;");
        
        _ = Task.Run(async () => 
        {
            await Task.Delay(3000);
            await InitializeSchema();
        });
    }

    private async Task InitializeSchema()
    {
        if (_schemaManager == null) return;

        try
        {
            Console.WriteLine("🔧 Checking QuestDB schema optimization...");
            var success = await _schemaManager.EnsureOptimizedSchemaExists();
            
            if (success)
            {
                var stats = await _schemaManager.GetTableStats();
                Console.WriteLine("📊 TelemetryTicks schema status:");
                foreach (var stat in stats)
                {
                    Console.WriteLine($"   {stat.Key}: {stat.Value}");
                }
            }
            else
            {
                Console.WriteLine("⚠️  Schema optimization failed, using existing table structure");
            }
        }
        catch (Exception ex)
        {
            Console.WriteLine($"⚠️  Schema initialization warning: {ex.Message}");
        }
    }

    public async Task<bool> TriggerSchemaOptimization()
    {
        if (_schemaManager == null)
        {
            Console.WriteLine("⚠️  Schema manager not initialized");
            return false;
        }

        try
        {
            Console.WriteLine("🔧 Manually triggering schema optimization...");
            var result = await _schemaManager.EnsureOptimizedSchemaExists();
            
            if (result)
            {
                var stats = await _schemaManager.GetTableStats();
                Console.WriteLine("📊 Schema optimization status:");
                foreach (var stat in stats)
                {
                    Console.WriteLine($"   {stat.Key}: {stat.Value}");
                }
            }
            
            return result;
        }
        catch (Exception ex)
        {
            Console.WriteLine($"❌ Manual schema optimization failed: {ex.Message}");
            return false;
        }
    }

    public void Dispose()
    {
        _sender?.Dispose();
        _schemaManager?.Dispose();
    }

    public async Task WriteBatch(TelemetryBatch? telData)
    {
        if (telData == null)
        {
            Console.WriteLine("No telemetry data to write");
            return;
        }

        if (_sender == null)
        {
            Console.WriteLine("ERROR: QuestDB sender is not initialized");
            return;
        }
        
        // Check circuit breaker
        if (_circuitBreakerOpen)
        {
            if (DateTime.UtcNow - _lastErrorTime > TimeSpan.FromMinutes(CircuitBreakerTimeoutMinutes))
            {
                _circuitBreakerOpen = false;
                _consecutiveErrors = 0;
                Console.WriteLine("🔄 Circuit breaker CLOSED - Resuming QuestDB writes");
            }
            else
            {
                Console.WriteLine("⛔ Circuit breaker OPEN - Skipping QuestDB write");
                return;
            }
        }

        try
        {
            foreach (var tel in telData.Records)
            {
                await _sender.Table("TelemetryTicks")
                    .Symbol("session_id", tel.SessionId)
                    .Symbol("track_name", tel.TrackName)
                    .Symbol("track_id", tel.TrackId)
                    
                    .Symbol("lap_id", tel.LapId ?? "unknown")
                    .Symbol("session_num", tel.SessionNum)
                    .Symbol("session_type", tel.SessionType ?? "Unknown")
                    .Symbol("session_name", tel.SessionName ?? "Unknown")
                    
                    .Column("car_id", tel.CarId)
                    
                    .Column("gear", tel.Gear)
                    .Column("player_car_position", (long)Math.Floor(tel.PlayerCarPosition))
                    
                    .Column("speed", tel.Speed)
                    .Column("lap_dist_pct", tel.LapDistPct)
                    .Column("session_time", tel.SessionTime)
                    .Column("lat", tel.Lat)
                    .Column("lon", tel.Lon)
                    .Column("lap_current_lap_time", tel.LapCurrentLapTime)
                    .Column("lapLastLapTime", tel.LapLastLapTime)
                    .Column("lapDeltaToBestLap", tel.LapDeltaToBestLap)
                    
                    .Column("throttle", (float)tel.Throttle)
                    .Column("brake", (float)tel.Brake)
                    .Column("steering_wheel_angle", (float)tel.SteeringWheelAngle)
                    .Column("rpm", (float)tel.Rpm)
                    .Column("velocity_x", (float)tel.VelocityX)
                    .Column("velocity_y", (float)tel.VelocityY)
                    .Column("velocity_z", (float)tel.VelocityZ)
                    .Column("fuel_level", (float)tel.FuelLevel)
                    .Column("alt", (float)tel.Alt)
                    .Column("lat_accel", (float)tel.LatAccel)
                    .Column("long_accel", (float)tel.LongAccel)
                    .Column("vert_accel", (float)tel.VertAccel)
                    .Column("pitch", (float)tel.Pitch)
                    .Column("roll", (float)tel.Roll)
                    .Column("yaw", (float)tel.Yaw)
                    .Column("yaw_north", (float)tel.YawNorth)
                    .Column("voltage", (float)tel.Voltage)
                    .Column("waterTemp", (float)tel.WaterTemp)
                    .Column("lFpressure", (float)tel.LFpressure)
                    .Column("rFpressure", (float)tel.RFpressure)
                    .Column("lRpressure", (float)tel.LRpressure)
                    .Column("rRpressure", (float)tel.RRpressure)
                    .Column("lFtempM", (float)tel.LFtempM)
                    .Column("rFtempM", (float)tel.RFtempM)
                    .Column("lRtempM", (float)tel.LRtempM)
                    .Column("rRtempM", (float)tel.RRtempM)
                    .AtAsync(tel.TickTime.ToDateTime());
            }
            
            await _sender.SendAsync();
            Console.WriteLine($"Successfully wrote {telData.Records.Count} telemetry points");
            
            // Reset error count on successful write
            _consecutiveErrors = 0;
        }
        catch (Exception ex)
        {
            Console.WriteLine($"ERROR writing to QuestDB: {ex.Message}");
            
            // Increment error count and check circuit breaker
            _consecutiveErrors++;
            _lastErrorTime = DateTime.UtcNow;
            
            if (_consecutiveErrors >= MaxConsecutiveErrors)
            {
                _circuitBreakerOpen = true;
                Console.WriteLine($"⛔ Circuit breaker OPENED after {_consecutiveErrors} consecutive errors - Stopping writes for {CircuitBreakerTimeoutMinutes} minutes");
            }
            
            try
            {
                _sender?.Dispose();
                string? url = Environment.GetEnvironmentVariable("QUESTDB_URL");
                if (url != null)
                {
                    _sender = Sender.New($"http::addr={url.Replace("http://", "")};auto_flush_rows=2000;auto_flush_interval=500;");
                    Console.WriteLine("🔄 QuestDB sender reset after error");
                }
            }
            catch (Exception resetEx)
            {
                Console.WriteLine($"⚠️  Failed to reset sender: {resetEx.Message}");
            }
        }
    }
}