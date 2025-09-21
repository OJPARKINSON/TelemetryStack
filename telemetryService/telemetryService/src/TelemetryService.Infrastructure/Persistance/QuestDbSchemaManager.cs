using System.Text.Json;
using QuestDB;
using QuestDB.Senders;

namespace TelemetryService.Infrastructure.Persistence;

public class QuestDbSchemaManager : IDisposable
{
    private readonly string _questDbUrl;
    private readonly HttpClient _httpClient;
    private volatile bool _disposed = false;

    public QuestDbSchemaManager(string questDbUrl)
    {
        _questDbUrl = questDbUrl.Replace("http://", "").Replace("https://", "");
        _httpClient = new HttpClient()
        {
            Timeout = TimeSpan.FromSeconds(60)
        };
    }

    public async Task<bool> EnsureOptimizedSchemaExists()
    {
        try
        {
            Console.WriteLine("🔧 Checking QuestDB schema optimization...");
            
            // Wait for QuestDB to be ready
            await WaitForQuestDbReady();
            
            // Clean up any orphaned tables first
            await CleanupOrphanedTables();
            
            var tableInfo = await GetTableInfo("TelemetryTicks");
            
            if (tableInfo == null)
            {
                Console.WriteLine("📋 No TelemetryTicks table found, creating optimized version...");
                await CreateOptimizedTable();
                await AddOptimizedIndexes();
                Console.WriteLine("✅ Optimized TelemetryTicks table created successfully");
                return true;
            }
            
            var needsOptimization = await NeedsOptimization(tableInfo);
            var needsIndexes = await NeedsCriticalIndexes();
            
            if (needsOptimization)
            {
                Console.WriteLine("🔄 Existing table detected, applying optimizations...");
                await OptimizeExistingTable();
                Console.WriteLine("✅ Table optimization completed");
                return true;
            }
            else if (needsIndexes)
            {
                Console.WriteLine("📊 Adding missing performance indexes...");
                await AddEssentialIndexes();
                Console.WriteLine("✅ Performance indexes added");
                return true;
            }
            else
            {
                Console.WriteLine("✅ TelemetryTicks table is already optimized");
                await LogCurrentTableStats();
                return true;
            }
        }
        catch (Exception ex)
        {
            Console.WriteLine($"⚠️  Schema management warning: {ex.Message}");
            Console.WriteLine("   Continuing with existing table structure...");
            return false;
        }
    }

    private async Task<JsonElement?> GetTableInfo(string tableName)
    {
        try
        {
            var query = $"SHOW COLUMNS FROM {tableName}";
            var response = await ExecuteQuery(query);
            return response;
        }
        catch
        {
            // Table doesn't exist
            return null;
        }
    }

    private async Task<bool> NeedsOptimization(JsonElement? tableInfo)
    {
        if (!tableInfo.HasValue || !tableInfo.Value.TryGetProperty("dataset", out var dataset))
            return false;

        foreach (var row in dataset.EnumerateArray())
        {
            var rowArray = row.EnumerateArray().ToArray();
            if (rowArray.Length >= 2)
            {
                var columnName = rowArray[0].GetString();
                var columnType = rowArray[1].GetString();
                
                if (columnName == "gear" && columnType == "SYMBOL")
                {
                    return true;
                }
            }
        }

        var hasSessionIndex = await HasIndex("session_id");
        var hasTrackIndex = await HasIndex("track_name");
        
        return !hasSessionIndex || !hasTrackIndex;
    }

    private async Task<bool> HasIndex(string columnName)
    {
        try
        {
            var query = $"SELECT indexed FROM table_columns('TelemetryTicks') WHERE column = '{columnName}'";
            var response = await ExecuteQuery(query);
            
            if (response?.TryGetProperty("dataset", out var dataset) == true && 
                dataset.EnumerateArray().Any())
            {
                var firstRow = dataset.EnumerateArray().First();
                var isIndexed = firstRow.EnumerateArray().First().GetBoolean();
                return isIndexed;
            }
        }
        catch
        {
            // Assume no index if query fails
        }
        
        return false;
    }

    private async Task<bool> NeedsCriticalIndexes()
    {
        try
        {
            // Check if essential indexes exist
            var hasSessionIndex = await HasIndex("session_id");
            var hasTrackIndex = await HasIndex("track_name");
            
            return !hasSessionIndex || !hasTrackIndex;
        }
        catch
        {
            return true;
        }
    }

    private async Task AddEssentialIndexes()
    {
        try
        {
            var hasSessionIndex = await HasIndex("session_id");
            var hasTrackIndex = await HasIndex("track_name");
            var hasTrackIdIndex = await HasIndex("track_id");

            if (!hasSessionIndex)
            {
                Console.WriteLine("   Adding session_id index...");
                await ExecuteQuery("ALTER TABLE TelemetryTicks ALTER COLUMN session_id ADD INDEX");
            }

            if (!hasTrackIndex)
            {
                Console.WriteLine("   Adding track_name index...");
                await ExecuteQuery("ALTER TABLE TelemetryTicks ALTER COLUMN track_name ADD INDEX");
            }

            if (!hasTrackIdIndex)
            {
                Console.WriteLine("   Adding track_id index...");
                await ExecuteQuery("ALTER TABLE TelemetryTicks ALTER COLUMN track_id ADD INDEX");
            }
        }
        catch (Exception ex)
        {
            Console.WriteLine($"   Warning: Could not add some indexes: {ex.Message}");
        }
    }

    private async Task LogCurrentTableStats()
    {
        try
        {
            var stats = await GetTableStats();
            Console.WriteLine("📊 TelemetryTicks table stats:");
            foreach (var stat in stats)
            {
                Console.WriteLine($"   {stat.Key}: {stat.Value}");
            }
        }
        catch (Exception ex)
        {
            Console.WriteLine($"   Could not retrieve table stats: {ex.Message}");
        }
    }

    private async Task CreateOptimizedTable()
    {
        var createTableSql = @"
            CREATE TABLE TelemetryTicks4 (
                session_id SYMBOL CAPACITY 50000 INDEX,
                track_name SYMBOL CAPACITY 100 INDEX,
                track_id SYMBOL CAPACITY 100 INDEX,
                lap_id SYMBOL CAPACITY 500,
                session_num SYMBOL CAPACITY 20,
                session_type SYMBOL CAPACITY 10 INDEX,
                session_name SYMBOL CAPACITY 50 INDEX,
                car_id SYMBOL CAPACITY 1000 INDEX,
                gear INT,
                player_car_position LONG,
                speed DOUBLE,
                lap_dist_pct DOUBLE,
                session_time DOUBLE,
                lat DOUBLE,
                lon DOUBLE,
                lap_current_lap_time DOUBLE,
                lapLastLapTime DOUBLE,
                lapDeltaToBestLap DOUBLE,
                throttle FLOAT,
                brake FLOAT,
                steering_wheel_angle FLOAT,
                rpm FLOAT,
                velocity_x FLOAT,
                velocity_y FLOAT,
                velocity_z FLOAT,
                fuel_level FLOAT,
                alt FLOAT,
                lat_accel FLOAT,
                long_accel FLOAT,
                vert_accel FLOAT,
                pitch FLOAT,
                roll FLOAT,
                yaw FLOAT,
                yaw_north FLOAT,
                voltage FLOAT,
                waterTemp FLOAT,
                lFpressure FLOAT,
                rFpressure FLOAT,
                lRpressure FLOAT,
                rRpressure FLOAT,
                lFtempM FLOAT,
                rFtempM FLOAT,
                lRtempM FLOAT,
                rRtempM FLOAT,
                timestamp TIMESTAMP
            ) TIMESTAMP(timestamp) PARTITION BY HOUR 
            WITH maxUncommittedRows=1000000, dedup_upsert_keys=(session_id, car_id, timestamp);
        ";

        await ExecuteQuery(createTableSql);
    }

    private async Task OptimizeExistingTable()
    {
        var timestamp = DateTime.Now.ToString("yyyyMMdd_HHmmss");
        Console.WriteLine($"🔄 Starting table optimization process...");

        try
        {
            var currentStats = await GetTableStats();
            if (currentStats.TryGetValue("row_count", out var rowCountObj))
            {
                Console.WriteLine($"   Current data: {rowCountObj} records");
            }

            Console.WriteLine("   Creating backup of existing table...");
            await ExecuteQuery($"RENAME TABLE TelemetryTicks TO TelemetryTicks_backup_{timestamp}");

            Console.WriteLine("   Creating optimized table structure...");
            await CreateOptimizedTable();

            var migrationSql = $@"
                INSERT INTO TelemetryTicks 
                SELECT 
                    session_id,
                    track_name,
                    track_id,
                    lap_id,
                    session_num,
                    COALESCE(session_type, 'Unknown') as session_type,
                    COALESCE(session_name, 'Unknown') as session_name,
                    car_id,
                    CASE 
                        WHEN gear = '1' THEN 1
                        WHEN gear = '2' THEN 2  
                        WHEN gear = '3' THEN 3
                        WHEN gear = '4' THEN 4
                        WHEN gear = '5' THEN 5
                        WHEN gear = '6' THEN 6
                        WHEN gear = '7' THEN 7
                        WHEN gear = '8' THEN 8
                        WHEN gear = 'R' THEN -1
                        WHEN gear = 'N' THEN 0
                        ELSE 0
                    END as gear,
                    player_car_position,
                    speed,
                    lap_dist_pct, 
                    session_time,
                    lat,
                    lon,
                    lap_current_lap_time,
                    lapLastLapTime,
                    lapDeltaToBestLap,
                    throttle,  -- No cast needed since both are FLOAT
                    brake,
                    steering_wheel_angle,
                    rpm,
                    velocity_x,
                    velocity_y,
                    velocity_z,
                    fuel_level,
                    alt,
                    lat_accel,
                    long_accel,
                    vert_accel,
                    pitch,
                    roll,
                    yaw,
                    yaw_north,
                    voltage,
                    waterTemp,
                    lFpressure,
                    rFpressure,
                    lRpressure,
                    rRpressure,
                    lFtempM,
                    rFtempM,
                    lRtempM,
                    rRtempM,
                    timestamp
                FROM TelemetryTicks_backup_{timestamp};
            ";

            await ExecuteQuery(migrationSql);

            await AddOptimizedIndexes();

            var newStats = await GetTableStats();
            if (newStats.TryGetValue("row_count", out var newRowCountObj))
            {
                Console.WriteLine($"   Migration verified: {newRowCountObj} records in optimized table");
            }


        }
        catch (Exception ex)
        {
            Console.WriteLine($"❌ Migration failed: {ex.Message}");

            // Attempt rollback
            try
            {
                Console.WriteLine("🔄 Attempting rollback...");
                await ExecuteQuery("DROP TABLE TelemetryTicks");
                await ExecuteQuery($"RENAME TABLE TelemetryTicks_backup_{timestamp} TO TelemetryTicks");
                Console.WriteLine("✅ Rollback successful, original table restored");
            }
            catch (Exception rollbackEx)
            {
                Console.WriteLine($"❌ Rollback failed: {rollbackEx.Message}");
                Console.WriteLine($"⚠️  Manual intervention required. Backup table: TelemetryTicks_backup_{timestamp}");
            }

            throw;
        }
    }

    private async Task AddOptimizedIndexes()
    {
        try
        {
            await ExecuteQuery("ALTER TABLE TelemetryTicks ADD INDEX session_lap_idx (session_id, lap_id)");
            await ExecuteQuery("ALTER TABLE TelemetryTicks ADD INDEX track_session_idx (track_name, session_id)");
            await ExecuteQuery("ALTER TABLE TelemetryTicks ADD INDEX session_time_idx (session_id, session_time)");

            await ExecuteQuery("ALTER TABLE TelemetryTicks DEDUP ENABLE UPSERT KEYS (car_id, timestamp)");

            
            Console.WriteLine("✅ Composite indexes added successfully");
        }
        catch (Exception ex)
        {
            Console.WriteLine($"⚠️  Warning: Could not add some indexes: {ex.Message}");
        }
    }

    private async Task<JsonElement?> ExecuteQuery(string query)
    {
        if (_disposed)
        {
            throw new ObjectDisposedException(nameof(QuestDbSchemaManager));
        }

        try
        {
            var encodedQuery = Uri.EscapeDataString(query);
            var url = $"http://{_questDbUrl}/exec?query={encodedQuery}";
            
            var response = await _httpClient.GetStringAsync(url);
            var jsonDoc = JsonDocument.Parse(response);
            
            // Check for errors
            if (jsonDoc.RootElement.TryGetProperty("error", out var error))
            {
                throw new Exception($"QuestDB Error: {error.GetString()}");
            }
            
            return jsonDoc.RootElement;
        }
        catch (Exception ex)
        {
            Console.WriteLine($"Query execution error: {ex.Message}");
            Console.WriteLine($"Query: {query}");
            throw;
        }
    }

    public async Task<Dictionary<string, object>> GetTableStats()
    {
        try
        {
            var stats = new Dictionary<string, object>();
            
            // Get basic table info from system tables (much faster than scanning data)
            var tableInfoQuery = "SELECT table_name FROM tables WHERE table_name = 'TelemetryTicks'";
            var tableResponse = await ExecuteQuery(tableInfoQuery);
            if (tableResponse?.TryGetProperty("dataset", out var tableDataset) == true && 
                tableDataset.EnumerateArray().Any())
            {
                stats["table_exists"] = true;
                
                // Get approximate row count from table metadata (instant)
                try
                {
                    var quickStatsQuery = "SELECT last(timestamp) as max_time FROM TelemetryTicks LIMIT 1";
                    var quickResponse = await ExecuteQuery(quickStatsQuery);
                    if (quickResponse?.TryGetProperty("dataset", out var quickDataset) == true && 
                        quickDataset.EnumerateArray().Any())
                    {
                        var maxTime = quickDataset.EnumerateArray().First().EnumerateArray().First().GetString();
                        stats["max_time"] = maxTime ?? "unknown";
                        stats["status"] = "optimized";
                    }
                    else
                    {
                        stats["status"] = "empty";
                    }
                }
                catch
                {
                    // If quick stats fail, just mark as available
                    stats["status"] = "available";
                }
            }
            else
            {
                stats["table_exists"] = false;
                stats["status"] = "missing";
            }
            
            return stats;
        }
        catch (Exception ex)
        {
            Console.WriteLine($"Error getting table stats: {ex.Message}");
            return new Dictionary<string, object> { ["error"] = ex.Message };
        }
    }

    public void Dispose()
    {
        if (_disposed) return;
        
        _httpClient?.Dispose();
        _disposed = true;
        
        GC.SuppressFinalize(this);
    }

    private async Task CleanupOrphanedTables()
    {
        try
        {
            Console.WriteLine("🧹 Checking for orphaned tables...");
            
            // Get list of all tables
            var tablesQuery = "SHOW TABLES";
            var response = await ExecuteQuery(tablesQuery);
            
            if (response?.TryGetProperty("dataset", out var dataset) == true)
            {
                var tablesToDelete = new List<string>();
                
                foreach (var row in dataset.EnumerateArray())
                {
                    var rowArray = row.EnumerateArray().ToArray();
                    if (rowArray.Length > 0)
                    {
                        var tableName = rowArray[0].GetString();
                        
                        // Check if this is an orphaned table (numeric names that look like session IDs)
                        if (!string.IsNullOrEmpty(tableName) && 
                            tableName != "TelemetryTicks" && 
                            IsOrphanedTable(tableName))
                        {
                            tablesToDelete.Add(tableName);
                        }
                    }
                }
                
                // Delete orphaned tables
                foreach (var tableName in tablesToDelete)
                {
                    try
                    {
                        Console.WriteLine($"   🗑️  Dropping orphaned table: {tableName}");
                        await ExecuteQuery($"DROP TABLE {tableName}");
                    }
                    catch (Exception ex)
                    {
                        Console.WriteLine($"   ⚠️  Could not drop table {tableName}: {ex.Message}");
                    }
                }
                
                if (tablesToDelete.Count > 0)
                {
                    Console.WriteLine($"✅ Cleaned up {tablesToDelete.Count} orphaned tables");
                }
                else
                {
                    Console.WriteLine("✅ No orphaned tables found");
                }
            }
        }
        catch (Exception ex)
        {
            Console.WriteLine($"⚠️  Could not cleanup orphaned tables: {ex.Message}");
        }
    }

    private async Task WaitForQuestDbReady()
    {
        for (int i = 0; i < 10; i++)
        {
            try
            {
                var healthCheck = await ExecuteQuery("SELECT 1 as health_check");
                if (healthCheck != null)
                {
                    Console.WriteLine("✅ QuestDB connection verified");
                    return;
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"⏳ Waiting for QuestDB to be ready (attempt {i + 1}/10): {ex.Message}");
                await Task.Delay(2000);
            }
        }
        
        throw new Exception("QuestDB is not responding after 10 attempts");
    }

    private static bool IsOrphanedTable(string tableName)
    {
        // Check if table name looks like a session ID (numeric) or backup table
        if (string.IsNullOrEmpty(tableName))
            return false;
            
        // Remove known good table names
        if (tableName == "TelemetryTicks")
            return false;
            
        // Check for numeric-only names (likely session IDs)
        if (long.TryParse(tableName, out _))
        {
            Console.WriteLine($"   Found numeric table name (likely session ID): {tableName}");
            return true;
        }
        
        // Check for backup tables from failed migrations
        if (tableName.StartsWith("TelemetryTicks_backup_"))
        {
            Console.WriteLine($"   Found backup table: {tableName}");
            return true;
        }
        
        return false;
    }
}