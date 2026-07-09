namespace Ingest.Ibt;

public delegate void FieldSetter(ref TelemetryTick t, ReadOnlySpan<byte> buf, int offset);

internal static class SetterFactory
{
    static IReadOnlyDictionary<string, FieldSetter> Build()
    {
        return new Dictionary<string, FieldSetter> { };
    }
}