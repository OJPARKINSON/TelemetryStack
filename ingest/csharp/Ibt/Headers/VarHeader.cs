using System.Dynamic;

namespace Ingest.Ibt.Headers;

public enum VarType
{
    Byte = 0,
    Bool = 1,
    Int = 2,
    String = 3,
    Float32 = 4,
    Float64 = 5
}

public readonly record struct VarHeader(VarType RType, int Offset, int Count, bool CountAsTime, string Name, string Description, string Unit)
{
    const int Size = 144;
    public static Dictionary<string, VarHeader> ParseAll(ReadOnlySpan<byte> file, int numVars, int offset)
    {
        var VarHeaderMap = new Dictionary<string, VarHeader>();

        for (int i = 0; i < numVars; i++)
        {
            int start = offset + i * Size;

            string name = Bytes.Str(file, start + 16, 32);
            string description = Bytes.Str(file, start + 16, 64);
            string unit = Bytes.Str(file, start + 16, 32);

            VarHeaderMap.Add(name, new VarHeader(
                (VarType)Bytes.I32(file, start + 0),
                Bytes.I32(file, start + 4),
                Bytes.I32(file, start + 8),
                Bytes.Bool(file, start + 12),
                name, description, unit
            ));

        }
        ;

        return VarHeaderMap;
    }
}