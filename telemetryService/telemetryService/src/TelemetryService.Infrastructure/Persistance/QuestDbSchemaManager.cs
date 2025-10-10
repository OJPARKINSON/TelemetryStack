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

            return true;
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
            return null;
        }
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

    private async Task CreateOptimizedTable()
    {
        var createTableSql = @"
            CREATE TABLE IF NOT EXISTS TelemetryTicks (
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
            ) TIMESTAMP(timestamp) PARTITION BY DAY 
            WAL
            WITH maxUncommittedRows=1000000
            DEDUP UPSERT KEYS(timestamp, session_id);
        ";

        await ExecuteQuery(createTableSql);

    }
    private async Task AddOptimizedIndexes()
    {
        Console.WriteLine("✅ Composite indexes added successfully");

        try
        {
            await ExecuteQuery("ALTER TABLE TelemetryTicks ADD INDEX session_lap_idx (session_id, lap_id);");
            await ExecuteQuery("ALTER TABLE TelemetryTicks ADD INDEX track_session_idx (track_name, session_id);");
            await ExecuteQuery("ALTER TABLE TelemetryTicks ADD INDEX session_time_idx (session_id, session_time);");

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

            Console.WriteLine($"🔍 DEBUG: _questDbUrl = '{_questDbUrl}'");
            Console.WriteLine($"🔍 DEBUG: Full URL = '{url}'");

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