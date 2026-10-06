namespace Ingest.Ibt.Headers;

public readonly record struct DiskHeader(long StartDate, double StartTime, double EndTime, int LapCount, int RecordCount)
{
    public const int Size = 32;
    const int Offset = TelemetryHeader.Size;
    DateTimeOffset SessionStart { get; }
    public static DiskHeader Parse(ReadOnlySpan<byte> b)
    {
        long startDate = Bytes.I64(b, 0);
        double startTime = Bytes.F64(b, 8);
        double endTime = Bytes.F64(b, 16);
        int lapCount = Bytes.I32(b, 24);
        int recordCount = Bytes.I32(b, 28);


        Console.WriteLine($"startDate: {DateOnly.FromDateTime(DateTimeOffset.FromUnixTimeSeconds(startDate).DateTime).ToLongDateString()}");
        Console.WriteLine($"startTime: {TimeSpan.FromSeconds(startTime).ToString("hh':'mm':'ss")}");
        Console.WriteLine($"endTime: {TimeSpan.FromSeconds(endTime).ToString("hh':'mm':'ss")}");
        Console.WriteLine($"lapCount: {lapCount}");
        Console.WriteLine($"recordCount: {recordCount}");
        return new DiskHeader(startDate, startTime, endTime, lapCount, recordCount);
    }
}