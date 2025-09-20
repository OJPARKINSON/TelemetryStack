using System.Net.Sockets;
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
    
    private string _tableName = "TelemetryTicks";
    
    // Configuration with defaults
    private readonly string _tcpHost;
    private readonly int _tcpPort;
    private readonly string _username;
    private readonly string _password;
    private readonly int _batchSize;
    private readonly int _autoFlushRows;
    private readonly int _autoFlushInterval;
    private readonly int _connectionTimeout;
    private readonly int _retryAttempts;
    private readonly int _retryDelay;

    public QuestDbService()
    {
        // Load configuration from environment variables with defaults
        _tcpHost = Environment.GetEnvironmentVariable("QUESTDB_TCP_HOST") ?? "questdb";
        _tcpPort = int.TryParse(Environment.GetEnvironmentVariable("QUESTDB_TCP_PORT"), out var port) ? port : 9000;
        _username = Environment.GetEnvironmentVariable("QUESTDB_TCP_USERNAME") ?? "admin";
        _password = Environment.GetEnvironmentVariable("QUESTDB_TCP_PASSWORD") ?? "quest";
        _batchSize = int.TryParse(Environment.GetEnvironmentVariable("QUESTDB_BATCH_SIZE"), out var batch) ? batch : 250;
        _autoFlushRows = int.TryParse(Environment.GetEnvironmentVariable("QUESTDB_AUTO_FLUSH_ROWS"), out var flushRows) ? flushRows : 500;
        _autoFlushInterval = int.TryParse(Environment.GetEnvironmentVariable("QUESTDB_AUTO_FLUSH_INTERVAL"), out var flushInterval) ? flushInterval : 2000;
        _connectionTimeout = int.TryParse(Environment.GetEnvironmentVariable("QUESTDB_CONNECTION_TIMEOUT"), out var timeout) ? timeout : 30000;
        _retryAttempts = int.TryParse(Environment.GetEnvironmentVariable("QUESTDB_RETRY_ATTEMPTS"), out var retries) ? retries : 3;
        _retryDelay = int.TryParse(Environment.GetEnvironmentVariable("QUESTDB_RETRY_DELAY"), out var delay) ? delay : 1000;

        string tcpConnectionString = $"{_tcpHost}:{_tcpPort}";
        string httpUrl = $"{_tcpHost}:9000";

        Console.WriteLine($"🔗 Initializing QuestDB connections with optimized settings:");
        Console.WriteLine($"   TCP Host: {_tcpHost}:{_tcpPort}");
        Console.WriteLine($"   Batch Size: {_batchSize} records");
        Console.WriteLine($"   Auto Flush: {_autoFlushRows} rows / {_autoFlushInterval}ms");
        Console.WriteLine($"   Connection Timeout: {_connectionTimeout}ms");
        Console.WriteLine($"   Retry Policy: {_retryAttempts} attempts with {_retryDelay}ms delay");

        try
        {
            _schemaManager = new QuestDbSchemaManager(httpUrl);
            _sender = CreateSender(tcpConnectionString);

            _ = Task.Run(async () =>
            {
                await Task.Delay(3000);
                await InitializeSchema();
            });
        }
        catch (Exception ex)
        {
            Console.WriteLine($"❌ Failed to initialize QuestDB service: {ex.Message}");
        }
    }

    private ISender CreateSender(string hostAndPort)
    {
        string connectionString = $"tcp::addr={hostAndPort};auto_flush_rows={_autoFlushRows};auto_flush_interval={_autoFlushInterval};";
        Console.WriteLine($"   Creating sender with: {connectionString}");
        return Sender.New(connectionString);
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

    private static string SanitizeString(string value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return "unknown";
        }
        
        // Remove or replace characters that can break line protocol
        return value
            .Replace(",", "_")     // Field separator
            .Replace(" ", "_")     // Space separator 
            .Replace("=", "_")     // Key-value separator
            .Replace("\n", "_")    // Line separator
            .Replace("\r", "_")    // Carriage return
            .Replace("\"", "_")    // Quotes
            .Replace("'", "_")     // Single quotes
            .Replace("\\", "_")    // Backslash
            .Trim();
    }

    private static bool IsConnectionError(Exception ex)
    {
        // Check if this is a connection-related error that warrants sender reset
        var message = ex.Message?.ToLower() ?? "";
        var innerMessage = ex.InnerException?.Message?.ToLower() ?? "";
        
        return ex is ObjectDisposedException ||
               ex is HttpRequestException ||
               ex is SocketException ||
               message.Contains("connection") ||
               message.Contains("timeout") ||
               message.Contains("network") ||
               message.Contains("broken pipe") ||
               message.Contains("socketerror") ||
               message.Contains("could not write data to server") ||
               innerMessage.Contains("httpclient") ||
               innerMessage.Contains("disposed") ||
               innerMessage.Contains("broken pipe") ||
               innerMessage.Contains("transport connection");
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

        // Implement batch chunking if the batch is too large
        if (telData.Records.Count > _batchSize)
        {
            Console.WriteLine($"🔄 Large batch detected ({telData.Records.Count} records). Chunking into batches of {_batchSize}...");
            
            var chunks = telData.Records
                .Select((record, index) => new { record, index })
                .GroupBy(x => x.index / _batchSize)
                .Select(g => g.Select(x => x.record).ToList())
                .ToList();

            var totalProcessed = 0;
            var startTime = DateTime.UtcNow;

            for (int i = 0; i < chunks.Count; i++)
            {
                var chunk = chunks[i];
                var chunkBatch = new TelemetryBatch();
                chunkBatch.Records.AddRange(chunk);
                
                Console.WriteLine($"📦 Processing chunk {i + 1}/{chunks.Count} ({chunk.Count} records)");
                
                try
                {
                    await WriteBatchChunk(senderToUse, chunkBatch);
                    totalProcessed += chunk.Count;
                    
                    // Small delay between chunks to prevent overwhelming the connection
                    if (i < chunks.Count - 1)
                    {
                        await Task.Delay(50);
                    }
                }
                catch (Exception ex)
                {
                    Console.WriteLine($"⚠️  Failed to write chunk {i + 1}: {ex.Message}");
                    // Continue with next chunk - don't fail the entire batch
                }
            }
            
            var duration = DateTime.UtcNow - startTime;
            Console.WriteLine($"✅ Chunked batch processing complete: {totalProcessed}/{telData.Records.Count} records in {duration.TotalMilliseconds:F1}ms");
            return;
        }

        // Process normal-sized batches directly
        try
        {
            Console.WriteLine($"🔄 Starting batch write to QuestDB: {telData.Records.Count} records");
            await WriteBatchChunk(senderToUse, telData);
        }
        catch (Exception ex)
        {
            await HandleWriteError(ex, telData);
        }
    }

    private async Task WriteBatchChunk(ISender senderToUse, TelemetryBatch telData)
    {
        var startTime = DateTime.UtcNow;
        var processedCount = 0;
        
        foreach (var tel in telData.Records)
            {
                // Validate required fields to prevent empty table names and invalid data
                var sessionId = SanitizeString(tel.SessionId);
                var trackName = SanitizeString(tel.TrackName);
                var trackId = SanitizeString(tel.TrackId);
                var lapId = SanitizeString(tel.LapId);
                var sessionNum = SanitizeString(tel.SessionNum);
                var sessionType = SanitizeString(tel.SessionType);
                var sessionName = SanitizeString(tel.SessionName);
                var carId = SanitizeString(tel.CarId);

                // Skip records with critical missing data
                if (sessionId == "unknown" && trackName == "unknown")
                {
                    Console.WriteLine("⚠️  Skipping telemetry record with missing session_id and track_name");
                    continue;
                }
                
                processedCount++;
                if (processedCount % 1000 == 0)
                {
                    Console.WriteLine($"   📊 Processed {processedCount}/{telData.Records.Count} records...");
                }
                
                 
                try
                {
                    senderToUse.Table(_tableName)
                        .Symbol("session_id", sessionId)
                        .Symbol("track_name", trackName)
                        .Symbol("track_id", trackId)
                        .Symbol("lap_id", lapId)
                        .Symbol("session_num", sessionNum)
                        .Symbol("session_type", sessionType)
                        .Symbol("session_name", sessionName)
                        
                        .Column("car_id", carId)
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
                catch (Exception rowEx)
                {
                    Console.WriteLine($"⚠️  Error adding record {processedCount} to batch: {rowEx.GetType().Name}: {rowEx.Message}");
                    
                    // If this is a connection error during auto-flush, re-throw to trigger sender reset
                    if (IsConnectionError(rowEx))
                    {
                        Console.WriteLine($"🔄 Connection error during row processing, will trigger sender reset: ", rowEx);
                        throw;
                    }

                    Console.Write("⚠️ on-connection errors, skip this record and continue: ", rowEx);
                    
                    // For non-connection errors, skip this record and continue
                continue;
                }
        }
        
        Console.WriteLine($"📤 Sending batch to QuestDB via TCP... ({processedCount} records processed)");
        await senderToUse.SendAsync();
        var duration = DateTime.UtcNow - startTime;
        Console.WriteLine($"✅ Successfully wrote {processedCount}/{telData.Records.Count} telemetry points in {duration.TotalMilliseconds:F1}ms");
    }

    private async Task HandleWriteError(Exception ex, TelemetryBatch telData)
    {
        if (ex is OutOfMemoryException)
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
            return;
        }
        
        Console.WriteLine($"❌ ERROR writing to QuestDB: {ex.GetType().Name}");
        Console.WriteLine($"   Message: {ex.Message}");
        Console.WriteLine($"   Batch Size: {telData.Records.Count} records");
        Console.WriteLine($"   Stack Trace: {ex.StackTrace}");
        
        if (ex.InnerException != null)
        {
            Console.WriteLine($"   Inner Exception: {ex.InnerException.GetType().Name}: {ex.InnerException.Message}");
        }
        
        // Only reset sender for specific errors that indicate connection issues
        var isConnError = IsConnectionError(ex);
        Console.WriteLine($"   Classified as connection error: {isConnError}");
        
        if (isConnError)
        {
            ResetSender();
        }
        else
        {
            Console.WriteLine("⚠️  Data format error - sender reset not attempted");
        }
    }

    private void ResetSender()
    {
        lock (_senderLock)
        {
            if (!_disposed)
            {
                try
                {
                    Console.WriteLine("🔄 Attempting to reset QuestDB sender due to connection error...");
                    Console.WriteLine($"   Previous sender state: {(_sender != null ? "Active" : "Null")}");
                    
                    _sender?.Dispose();
                    _sender = null;
                    
                    string tcpConnectionString = $"{_tcpHost}:{_tcpPort}";
                    Console.WriteLine($"   Connecting to: {tcpConnectionString}");
                    
                    _sender = CreateSender(tcpConnectionString);
                    Console.WriteLine("✅ QuestDB sender reset successful");
                }
                catch (Exception resetEx)
                {
                    Console.WriteLine($"❌ Failed to reset sender: {resetEx.Message}");
                    _sender = null;
                }
            }
        }
    }
}