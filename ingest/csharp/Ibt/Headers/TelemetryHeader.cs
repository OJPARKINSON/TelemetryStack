namespace Ingest.Ibt.Headers;

public readonly record struct TelemetryHeader(int Version, int Status, int TickRate, int SessionInfoUpdate,
    int SessionInfoLength, int SessionInfoOffset, int NumVars,
    int VarHeaderOffset, int NumBuf, int BufLen, int BufOffset)
{
    public const int Size = 112;

    public static TelemetryHeader Parse(ReadOnlySpan<byte> b)
    {
        var header = new TelemetryHeader(
            Version: Bytes.I32(b, 0),
            Status: Bytes.I32(b, 4),
            TickRate: Bytes.I32(b, 8),
            SessionInfoUpdate: Bytes.I32(b, 12),
            SessionInfoLength: Bytes.I32(b, 16),
            SessionInfoOffset: Bytes.I32(b, 20),
            NumVars: Bytes.I32(b, 24),
            VarHeaderOffset: Bytes.I32(b, 28),
            NumBuf: Bytes.I32(b, 32),
            BufLen: Bytes.I32(b, 36),
            BufOffset: Bytes.I32(b, 52));

        if (header.Version != 2 || header.Status is < 0 or > 1 || header.TickRate is < 60 or > 360)
            throw new InvalidDataException($"invalid telemetry header: {header}");
        return header;
    }
}