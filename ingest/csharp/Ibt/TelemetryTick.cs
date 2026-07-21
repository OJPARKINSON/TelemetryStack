namespace Ingest.Ibt;

public struct TelemetryTick
{
    [Ibt("Speed")] public double speed;
    [Ibt("Gear")] public uint gear;
    [Ibt("Throttle")] public double throttle;
    [Ibt("Brake")] public double brake;
    [Ibt("RPM")] public double rpm;
    [Ibt("SteeringWheelAngle")] public double SteeringWheelAngle;
    [Ibt("LapDistPct")] public double lapDistPct;
    [Ibt("SessionTime")] public double sessionTime;
    [Ibt("Lat")] public double lat;
    [Ibt("Lon")] public double lon;
    [Ibt("LapId")] public string lapId;
    [Ibt("SessionNum")] public int sessionNum;

}