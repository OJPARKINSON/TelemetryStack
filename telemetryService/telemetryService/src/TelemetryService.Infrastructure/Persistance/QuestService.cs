using QuestDB;
using QuestDB.Senders;
using TelemetryService.Domain.Models;

namespace TelemetryService.Infrastructure.Persistence;

public class QuestDbService : IDisposable
{
    private ISender? _sender;
    private QuestDbSchemaManager? _schemaManager;
    private readonly object _senderLock = new object();
    private volatile bool _disposed = false;

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
        if (_disposed) return;
        
        lock (_senderLock)
        {
            if (_disposed) return;
            
            _sender?.Dispose();
            _schemaManager?.Dispose();
            _disposed = true;
        }
        
        GC.SuppressFinalize(this);
    }

    private static double GetValidDouble(double value)
    {
        if (double.IsNaN(value) || double.IsInfinity(value) || value == double.MinValue || value == double.MaxValue)
        {
            return 0.0;
        }
        return value;
    }

    public async Task WriteBatch(TelemetryBatch? telData)
    {
        if (_disposed)
        {
            Console.WriteLine("ERROR: QuestDB service has been disposed");
            return;
        }

        if (telData == null)
        {
            Console.WriteLine("No telemetry data to write");
            return;
        }

        ISender? senderToUse;
        lock (_senderLock)
        {
            if (_disposed)
            {
                Console.WriteLine("ERROR: QuestDB service has been disposed");
                return;
            }

            if (_sender == null)
            {
                Console.WriteLine("ERROR: QuestDB sender is not initialized");
                return;
            }
            
            senderToUse = _sender;
        }

        try
        {
            foreach (var tel in telData.Records)
            {
                senderToUse.Table("TelemetryTicks")
                    .Symbol("session_id", tel.SessionId ?? "unknown")
                    .Symbol("track_name", tel.TrackName ?? "unknown")
                    .Symbol("track_id", tel.TrackId ?? "unknown")
                    .Symbol("lap_id", tel.LapId ?? "unknown")
                    .Symbol("session_num", tel.SessionNum ?? "0")
                    .Symbol("session_type", tel.SessionType ?? "Unknown")
                    .Symbol("session_name", tel.SessionName ?? "Unknown")
                    
                    .Column("car_id", tel.CarId ?? "unknown")
                    .Column("gear", tel.Gear)
                    .Column("player_car_position", (long)Math.Max(0, Math.Floor(tel.PlayerCarPosition)))
                    
                    .Column("speed", GetValidDouble(tel.Speed))
                    .Column("lap_dist_pct", GetValidDouble(tel.LapDistPct))
                    .Column("session_time", GetValidDouble(tel.SessionTime))
                    .Column("lat", GetValidDouble(tel.Lat))
                    .Column("lon", GetValidDouble(tel.Lon))
                    .Column("lap_current_lap_time", GetValidDouble(tel.LapCurrentLapTime))
                    .Column("lapLastLapTime", GetValidDouble(tel.LapLastLapTime))
                    .Column("lapDeltaToBestLap", GetValidDouble(tel.LapDeltaToBestLap))
                    
                    .Column("throttle", (float)GetValidDouble(tel.Throttle))
                    .Column("brake", (float)GetValidDouble(tel.Brake))
                    .Column("steering_wheel_angle", (float)GetValidDouble(tel.SteeringWheelAngle))
                    .Column("rpm", (float)GetValidDouble(tel.Rpm))
                    .Column("velocity_x", (float)GetValidDouble(tel.VelocityX))
                    .Column("velocity_y", (float)GetValidDouble(tel.VelocityY))
                    .Column("velocity_z", (float)GetValidDouble(tel.VelocityZ))
                    .Column("fuel_level", (float)GetValidDouble(tel.FuelLevel))
                    .Column("alt", (float)GetValidDouble(tel.Alt))
                    .Column("lat_accel", (float)GetValidDouble(tel.LatAccel))
                    .Column("long_accel", (float)GetValidDouble(tel.LongAccel))
                    .Column("vert_accel", (float)GetValidDouble(tel.VertAccel))
                    .Column("pitch", (float)GetValidDouble(tel.Pitch))
                    .Column("roll", (float)GetValidDouble(tel.Roll))
                    .Column("yaw", (float)GetValidDouble(tel.Yaw))
                    .Column("yaw_north", (float)GetValidDouble(tel.YawNorth))
                    .Column("voltage", (float)GetValidDouble(tel.Voltage))
                    .Column("waterTemp", (float)GetValidDouble(tel.WaterTemp))
                    .Column("lFpressure", (float)GetValidDouble(tel.LFpressure))
                    .Column("rFpressure", (float)GetValidDouble(tel.RFpressure))
                    .Column("lRpressure", (float)GetValidDouble(tel.LRpressure))
                    .Column("rRpressure", (float)GetValidDouble(tel.RRpressure))
                    .Column("lFtempM", (float)GetValidDouble(tel.LFtempM))
                    .Column("rFtempM", (float)GetValidDouble(tel.RFtempM))
                    .Column("lRtempM", (float)GetValidDouble(tel.LRtempM))
                    .Column("rRtempM", (float)GetValidDouble(tel.RRtempM))
                    .At(tel.TickTime.ToDateTime());
            }
            
            await senderToUse.SendAsync();
            Console.WriteLine($"Successfully wrote {telData.Records.Count} telemetry points");
        }
        catch (OutOfMemoryException ex)
        {
            Console.WriteLine($"❌ CRITICAL: OutOfMemoryException in QuestDB write operation");
            Console.WriteLine($"   Message: {ex.Message}");
            Console.WriteLine($"   Stack Trace: {ex.StackTrace}");
            Console.WriteLine($"   Batch Size: {telData.Records.Count} records");
            
            // Log memory state
            var process = System.Diagnostics.Process.GetCurrentProcess();
            var memoryUsageGB = (double)process.WorkingSet64 / (1024 * 1024 * 1024);
            var gcMemoryMB = GC.GetTotalMemory(false) / (1024 * 1024);
            Console.WriteLine($"   Current Memory Usage: {memoryUsageGB:F2}GB (Working Set)");
            Console.WriteLine($"   GC Memory: {gcMemoryMB:F2}MB");
        }
        catch (Exception ex)
        {
            Console.WriteLine($"❌ ERROR writing to QuestDB: {ex.GetType().Name}");
            Console.WriteLine($"   Message: {ex.Message}");
            Console.WriteLine($"   Stack Trace: {ex.StackTrace}");
            Console.WriteLine($"   Batch Size: {telData.Records.Count} records");
            
            if (ex.InnerException != null)
            {
                Console.WriteLine($"   Inner Exception: {ex.InnerException.GetType().Name}: {ex.InnerException.Message}");
            }
            
            lock (_senderLock)
            {
                if (!_disposed)
                {
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
    }
}