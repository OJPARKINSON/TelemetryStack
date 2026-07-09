namespace Ingest.Ibt;

public struct TelemetryTick
{
    [Ibt("Speed")] public double speed;
    [Ibt("Gear")] public uint Gear;
    [Ibt("SessionTime")] public double SessionTime;

}