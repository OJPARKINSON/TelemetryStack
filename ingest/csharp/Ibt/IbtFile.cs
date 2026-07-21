using System.IO.MemoryMappedFiles;

namespace Ingest.Ibt;

public sealed class IbtFile : IDisposable
{
    private readonly MemoryMappedFile _mmf;
    private readonly MemoryMappedViewAccessor _view;
    private readonly unsafe byte* _ptr;
    private readonly long _length;

    public unsafe IbtFile(string path)
    {
        _length = new FileInfo(path).Length;
        _mmf = MemoryMappedFile.CreateFromFile(path, FileMode.Open, mapName: null, capacity: 0, MemoryMappedFileAccess.Read);
        _view = _mmf.CreateViewAccessor(0, 0, MemoryMappedFileAccess.Read);
        byte* p = null;
        _view.SafeMemoryMappedViewHandle.AcquirePointer(ref p);
        _ptr = p + _view.PointerOffset;
    }

    public unsafe ReadOnlySpan<byte> Span => new(_ptr, (int)_length);

    void IDisposable.Dispose()
    {
        _view.SafeMemoryMappedViewHandle.ReleasePointer();
        _view.Dispose();
        _mmf.Dispose();
    }
}