package persistance

import (
	"context"
	"fmt"
	"time"

	"github.com/ojparkinson/telemetryService/internal/messaging"
	qdb "github.com/questdb/go-questdb-client/v4"
)

func SaveTickToDB(record *messaging.Telemetry) {
	ctx := context.TODO()

	client, err := qdb.NewLineSender(ctx, qdb.WithHttp(), qdb.WithAddress("localhost:9000"), qdb.WithAutoFlushRows(10000), qdb.WithRequestTimeout(60*time.Second)) //, qdb.WithBasicAuth("admin", "quest"
	if err != nil {
		fmt.Println("Failed to connect to DB: ", err)
	}

	client.Table("TelemetryTicks").
		Symbol("session_id", record.SessionId).
		Symbol("track_name", record.TrackName).
		Symbol("track_id", record.TrackId).
		Symbol("lap_id", record.LapId).
		Symbol("session_num", record.SessionNum).
		Symbol("session_type", record.SessionType).
		Symbol("session_name", record.SessionName).
		StringColumn("car_id", record.CarId).
		Int64Column("gear", int64(record.Gear)).
		Int64Column("player_car_position", int64(record.PlayerCarPosition)).
		Float64Column("speed", record.Speed).
		Float64Column("lap_dist_pct", record.LapDistPct).
		Float64Column("session_time", record.SessionTime).
		Float64Column("lat", record.Lat).
		Float64Column("lon", record.Lon).
		Float64Column("lap_current_lap_time", record.LapCurrentLapTime).
		Float64Column("lapLastLapTime", record.LapLastLapTime).
		Float64Column("lapDeltaToBestLap", record.LapDeltaToBestLap).
		Float64Column("throttle", record.Throttle).
		Float64Column("brake", record.Brake).
		Float64Column("steering_wheel_angle", record.SteeringWheelAngle).
		Float64Column("rpm", record.Rpm).
		Float64Column("velocity_x", record.VelocityX).
		Float64Column("velocity_y", record.VelocityY).
		Float64Column("velocity_z", record.VelocityZ).
		Float64Column("fuel_level", record.FuelLevel).
		Float64Column("alt", record.Alt).
		Float64Column("lat_accel", record.LatAccel).
		Float64Column("long_accel", record.LongAccel).
		Float64Column("vert_accel", record.VertAccel).
		Float64Column("pitch", record.Pitch).
		Float64Column("roll", record.Roll).
		Float64Column("yaw", record.Yaw).
		Float64Column("yaw_north", record.YawNorth).
		Float64Column("voltage", record.Voltage).
		Float64Column("waterTemp", record.WaterTemp).
		Float64Column("lFpressure", record.LFpressure).
		Float64Column("rFpressure", record.RFpressure).
		Float64Column("lRpressure", record.LRpressure).
		Float64Column("rRpressure", record.RRpressure).
		Float64Column("lFtempM", record.LFtempM).
		Float64Column("rFtempM", record.RFtempM).
		Float64Column("lRtempM", record.LRtempM).
		Float64Column("rRtempM", record.RRtempM).
		At(ctx, record.TickTime.AsTime())

	if err != nil {
		panic("Failed to insert data")
	}

	err = client.Flush(ctx)
	if err != nil {
		fmt.Println(err)
		panic("Failed to flush data: ")
	}
}
