using Ingest.Ibt.Headers;

namespace Ingest.Ibt;

internal readonly record struct VarSetter(int Offset, FieldSetter Set);

public sealed class TickParser
{
    public TickParser(IbtHeaders headers, IEnumerable<string>? whitelist = null) { }
    public bool Next(ref TelemetryTick tick, ReadOnlySpan<byte> file)
    {
        return false;
    }

    int RecordCount { get; }
}