namespace Ingest.Ibt;

[AttributeUsage(AttributeTargets.Field)]
public sealed class IbtAttribute(string name) : Attribute
{

    public string Name { get; } = name;
}