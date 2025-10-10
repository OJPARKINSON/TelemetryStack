using QuestDB.Senders;
using TelemetryService.Domain.Models;

namespace TelemetryService.Infrastructure.Persistence;

public static class QuestDbService
{
    private const int PartitionCount = 4;
    private const int MaxRetries = 3;

    public static async Task WriteBatch(List<Telemetry> records, string tableName = "TelemetryTicks")
    {
        if (records == null || !records.Any())
        {
            Console.WriteLine("⚠️  Empty batch received, skipping");
            return;
        }

        var retryCount = 0;

        while (retryCount <= MaxRetries)
        {
            try
            {
                await WritePartitionedBatch(records, tableName);
                return;
            }
            catch (Exception ex) when (IsRetryableError(ex) && retryCount < MaxRetries)
            {
                retryCount++;
                var delay = Math.Min(1000 * retryCount, 5000);
                Console.WriteLine($"⚠️  Attempt {retryCount} failed, retrying in {delay}ms: {ex.Message}");
                await Task.Delay(delay);
            }
        }

        Console.WriteLine($"❌ Failed to write batch after {MaxRetries + 1} attempts");
        throw new InvalidOperationException($"Failed to write batch after {MaxRetries + 1} attempts");
    }

    private static async Task WritePartitionedBatch(List<Telemetry> records, string tableName)
    {
        // Partition records by session_id hash for consistent routing (dedup safety)
        var partitions = records
            .GroupBy(r => Math.Abs((r.SessionId ?? "unknown").GetHashCode()) % PartitionCount)
            .ToList();

        Console.WriteLine($"📊 Partitioned {records.Count} records into {partitions.Count} buckets");

        var senders = new List<ISender>(PartitionCount);
        try
        {
            for (int i = 0; i < PartitionCount; i++)
            {
                senders.Add(QuestDbSenderPool.Get());
            }

            var writeTasks = partitions.Select(async (partition, index) =>
            {
                var partitionRecords = partition.ToList();
                var sender = senders[index % senders.Count];

                try
                {
                    await WritePartition(sender, partitionRecords, tableName, index);
                }
                catch (Exception ex)
                {
                    Console.WriteLine($"❌ Partition {index} failed: {ex.Message}");
                    throw;
                }
            });

            await Task.WhenAll(writeTasks);

            Console.WriteLine($"✅ Successfully wrote {records.Count} records across {partitions.Count} partitions");
        }
        finally
        {
            // Return all senders to pool
            foreach (var sender in senders)
            {
                QuestDbSenderPool.Return(sender);
            }
        }
    }

    private static async Task WritePartition(ISender sender, List<Telemetry> records, string tableName, int partitionIndex)
    {
        var validRecords = records.Where(IsValidRecord).ToList();

        if (!validRecords.Any())
        {
            return;
        }

        foreach (var record in validRecords)
        {
            sender.Table(tableName)
                .Symbol("session_id", Sanitize(record.SessionId))
                .Symbol("track_name", Sanitize(record.TrackName))
                .Symbol("track_id", Sanitize(record.TrackId))
                .Symbol("lap_id", Sanitize(record.LapId))
                .Symbol("session_num", Sanitize(record.SessionNum))
                .Symbol("session_type", Sanitize(record.SessionType))
                .Symbol("session_name", Sanitize(record.SessionName))

                .Column("car_id", Sanitize(record.CarId))
                .Column("gear", ValidateInt(record.Gear))
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
                .At(record.TickTime.ToDateTime());
        }

        await sender.SendAsync();

        Console.WriteLine($"  📦 Partition {partitionIndex}: {validRecords.Count} records");
    }

    #region Validation Helpers

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

    private static string Sanitize(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
            return "unknown";

        var length = value.Length;
        Span<char> buffer = length <= 256 ? stackalloc char[length] : new char[length];

        value.AsSpan().CopyTo(buffer);

        for (int i = 0; i < buffer.Length; i++)
        {
            switch (buffer[i])
            {
                case ',':
                case ' ':
                case '=':
                case '\n':
                case '\r':
                case '"':
                case '\'':
                case '\\':
                    buffer[i] = '_';
                    break;
            }
        }

        var trimmed = buffer.Trim();
        return new string(trimmed);
    }

    private static double ValidateDouble(double value)
    {
        return double.IsNaN(value) || double.IsInfinity(value) ||
               value == double.MinValue || value == double.MaxValue ? 0.0 : value;
    }

    private static int ValidateInt(uint value)
    {
        // Handle iRacing's invalid sentinel value (4294967295 = uint.MaxValue)
        // QuestDB INT is signed 32-bit with max value 2,147,483,647
        if (value == uint.MaxValue || value > int.MaxValue)
        {
            return 0;
        }
        return (int)value;
    }

    public static bool IsValidRecord(Telemetry record)
    {
        return !string.IsNullOrWhiteSpace(record.SessionId) ||
               !string.IsNullOrWhiteSpace(record.TrackName);
    }

    #endregion
}
