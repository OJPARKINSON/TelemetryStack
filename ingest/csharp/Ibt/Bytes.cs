using System.Buffers.Binary;
using System.Text;

namespace Ingest.Ibt;

internal static class Bytes
{
    private const uint InvalidValue = 0xFFFF_FFFF;

    public static int I32(ReadOnlySpan<byte> b, int o)
    {
        uint v = BinaryPrimitives.ReadUInt32LittleEndian(b.Slice(o, 4));
        return v == InvalidValue ? -1 : (int)v;
    }

    public static long I64(ReadOnlySpan<byte> b, int o) => BinaryPrimitives.ReadInt64LittleEndian(b.Slice(o, 8));
    public static float F32(ReadOnlySpan<byte> b, int o) => BinaryPrimitives.ReadSingleLittleEndian(b.Slice(o, 4));
    public static double F64(ReadOnlySpan<byte> b, int o) => BinaryPrimitives.ReadDoubleLittleEndian(b.Slice(o, 8));
    public static bool Bool(ReadOnlySpan<byte> b, int o) => b[o] != 0;

    public static string Str(ReadOnlySpan<byte> b, int o, int len) => Str(b.Slice(o, len));
    static string Str(ReadOnlySpan<byte> b)
    {
        int n = b.IndexOf((byte)0);
        return Encoding.ASCII.GetString(n < 0 ? b : b[..n]); // trims null padding
    }
}