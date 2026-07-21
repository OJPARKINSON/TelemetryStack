namespace Ingest.Ibt.Headers;

public sealed record IbtHeaders(
    TelemetryHeader Telemetry,
    DiskHeader Disk,
    IReadOnlyDictionary<string, VarHeader> Vars,
    VarBuffer[] Buffers)
{
    public static IbtHeaders Parse(ReadOnlySpan<byte> file)
    {
        Console.WriteLine($"file size: {file.Length}");
        var telemetry = TelemetryHeader.Parse(file.Slice(0, TelemetryHeader.Size));
        var disk = DiskHeader.Parse(file.Slice(TelemetryHeader.Size, DiskHeader.Size));
        var vars = VarHeader.ParseAll(file, telemetry.NumVars, telemetry.VarHeaderOffset);
        var buffers = VarBuffer.ParseAll(file, telemetry.NumBuf);

        return new IbtHeaders(telemetry, disk, vars, buffers);
    }
    IReadOnlyCollection<string> AvailableVars = (IReadOnlyCollection<string>)Vars.Keys;
};

