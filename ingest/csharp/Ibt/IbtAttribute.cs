namespace Ingest.Ibt;

[AttributeUsage(AttributeTargets.Field)]
public sealed class IbtAttribute(string name) : Attribute
{
    string Name { get; }
}