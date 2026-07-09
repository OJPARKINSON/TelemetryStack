namespace Ingest.Ibt;

public sealed class IbtFile : IDisposable
{
    private readonly byte[] _data;
    public IbtFile(string path) => _data = File.ReadAllBytes(path);
    public ReadOnlySpan<byte> Span => _data;
    public int Length => _data.Length;
    void IDisposable.Dispose()
    {
        throw new NotImplementedException();
    }
}