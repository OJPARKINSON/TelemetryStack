namespace Ingest.Ibt.Headers;

public readonly record struct VarBuffer(int TickCount, int bufOffset)
{
    const int BaseOffset = 48;
    const int Increment = 16;
    public static VarBuffer[] ParseAll(ReadOnlySpan<byte> file, int numBuf)
    {
        return [new VarBuffer()];
    }
}