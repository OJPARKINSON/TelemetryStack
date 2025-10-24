package persistance

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/ojparkinson/telemetryService/internal/messaging"
	qdb "github.com/questdb/go-questdb-client/v4"
)

func WriteBatch(sender qdb.LineSender, records []*messaging.Telemetry) error {
	ctx := context.Background()

	for _, record := range records {
		sender.Table("TelemetryTicks").
			Symbol("session_id", sanitise(record.SessionId)).
			Symbol("track_name", sanitise(record.TrackName)).
			Symbol("track_id", sanitise(record.TrackId)).
			Symbol("lap_id", sanitise(record.LapId)).
			Symbol("session_num", sanitise(record.SessionNum)).
			Symbol("session_type", sanitise(record.SessionType)).
			Symbol("session_name", sanitise(record.SessionName)).
			StringColumn("car_id", sanitise(record.CarId)).
			Int64Column("gear", validateInt(record.Gear)).
			Int64Column("player_car_position", validateInt(record.PlayerCarPosition)).
			Float64Column("speed", validateDouble(record.Speed)).
			Float64Column("lap_dist_pct", validateDouble(record.LapDistPct)).
			Float64Column("session_time", validateDouble(record.SessionTime)).
			Float64Column("lat", validateDouble(record.Lat)).
			Float64Column("lon", validateDouble(record.Lon)).
			Float64Column("lap_current_lap_time", validateDouble(record.LapCurrentLapTime)).
			Float64Column("lapLastLapTime", validateDouble(record.LapLastLapTime)).
			Float64Column("lapDeltaToBestLap", validateDouble(record.LapDeltaToBestLap)).
			Float64Column("throttle", validateDouble(record.Throttle)).
			Float64Column("brake", validateDouble(record.Brake)).
			Float64Column("steering_wheel_angle", validateDouble(record.SteeringWheelAngle)).
			Float64Column("rpm", validateDouble(record.Rpm)).
			Float64Column("velocity_x", validateDouble(record.VelocityX)).
			Float64Column("velocity_y", validateDouble(record.VelocityY)).
			Float64Column("velocity_z", validateDouble(record.VelocityZ)).
			Float64Column("fuel_level", validateDouble(record.FuelLevel)).
			Float64Column("alt", validateDouble(record.Alt)).
			Float64Column("lat_accel", validateDouble(record.LatAccel)).
			Float64Column("long_accel", validateDouble(record.LongAccel)).
			Float64Column("vert_accel", validateDouble(record.VertAccel)).
			Float64Column("pitch", validateDouble(record.Pitch)).
			Float64Column("roll", validateDouble(record.Roll)).
			Float64Column("yaw", validateDouble(record.Yaw)).
			Float64Column("yaw_north", validateDouble(record.YawNorth)).
			Float64Column("voltage", validateDouble(record.Voltage)).
			Float64Column("waterTemp", validateDouble(record.WaterTemp)).
			Float64Column("lFpressure", validateDouble(record.LFpressure)).
			Float64Column("rFpressure", validateDouble(record.RFpressure)).
			Float64Column("lRpressure", validateDouble(record.LRpressure)).
			Float64Column("rRpressure", validateDouble(record.RRpressure)).
			Float64Column("lFtempM", validateDouble(record.LFtempM)).
			Float64Column("rFtempM", validateDouble(record.RFtempM)).
			Float64Column("lRtempM", validateDouble(record.LRtempM)).
			Float64Column("rRtempM", validateDouble(record.RRtempM)).
			At(ctx, record.TickTime.AsTime())
	}

	err := sender.Flush(ctx)
	if err != nil {
		return fmt.Errorf("flush failed: %w", err)
	}

	fmt.Printf("wrote %d records to QuestDb\n", len(records))
	return nil
}

func sanitise(value string) string {
	if value == "" {
		return "unkown"
	}

	value = strings.ReplaceAll(value, ",", "_")
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "=", "_")
	value = strings.ReplaceAll(value, "\n", "_")
	value = strings.ReplaceAll(value, "\r", "_")
	value = strings.ReplaceAll(value, "\"", "_")
	value = strings.ReplaceAll(value, "'", "_")
	value = strings.ReplaceAll(value, "\\", "_")

	return strings.TrimSpace(value)
}

func validateDouble(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0.0
	}
	if value == math.MaxFloat64 || value == -math.MaxFloat64 {
		return 0.0
	}
	return value
}

func validateInt(value uint32) int64 {
	// Handle iRacing's invalid sentinel value (4294967295 = uint.MaxValue)
	if value == 0xFFFFFFFF {
		return 0
	}
	return int64(value)
}
