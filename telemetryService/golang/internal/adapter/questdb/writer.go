package questdb

import (
	"context"
	"log"
	"time"

	"github.com/ojparkinson/telemetryService/internal/messaging"
)

func (r *Repository) WriteBatch(ctx context.Context, records []*messaging.Telemetry) error {
	if len(records) == 0 {
		return nil
	}

	sender := r.writers.Get()
	defer r.writers.Return(sender)

	for _, rec := range records {
		ts := time.Now()

		if rec.TickTime != nil {
			ts = rec.TickTime.AsTime()
		}

		err := sender.Table("TelemetryTicks").Symbol("session_id", rec.SessionId).
			Symbol("track_name", rec.TrackName).
			Symbol("track_id", rec.TrackId).
			Symbol("lap_id", rec.LapId).
			Symbol("session_num", rec.SessionNum).
			Symbol("session_type", rec.SessionType).
			Symbol("session_name", rec.SessionName).
			Symbol("car_id", rec.CarId).
			Int64Column("gear", int64(rec.Gear)).
			Int64Column("player_car_position", int64(rec.PlayerCarPosition)).
			Float64Column("speed", rec.Speed).
			Float64Column("lap_dist_pct", rec.LapDistPct).
			Float64Column("session_time", rec.SessionTime).
			Float64Column("lat", rec.Lat).
			Float64Column("lon", rec.Lon).
			Float64Column("lap_current_lap_time", rec.LapCurrentLapTime).
			Float64Column("lapLastLapTime", rec.LapLastLapTime).
			Float64Column("lapDeltaToBestLap", rec.LapDeltaToBestLap).
			Float64Column("throttle", rec.Throttle).
			Float64Column("brake", rec.Brake).
			Float64Column("steering_wheel_angle", rec.SteeringWheelAngle).
			Float64Column("rpm", rec.Rpm).
			Float64Column("velocity_x", rec.VelocityX).
			Float64Column("velocity_y", rec.VelocityY).
			Float64Column("velocity_z", rec.VelocityZ).
			Float64Column("fuel_level", rec.FuelLevel).
			Float64Column("alt", rec.Alt).
			Float64Column("lat_accel", rec.LatAccel).
			Float64Column("long_accel", rec.LongAccel).
			Float64Column("vert_accel", rec.VertAccel).
			Float64Column("pitch", rec.Pitch).
			Float64Column("roll", rec.Roll).
			Float64Column("yaw", rec.Yaw).
			Float64Column("yaw_north", rec.YawNorth).
			Float64Column("voltage", rec.Voltage).
			Float64Column("waterTemp", rec.WaterTemp).
			Float64Column("lFpressure", rec.LFpressure).
			Float64Column("rFpressure", rec.RFpressure).
			Float64Column("lRpressure", rec.LRpressure).
			Float64Column("rRpressure", rec.RRpressure).
			Float64Column("lFtempM", rec.LFtempM).
			Float64Column("rFtempM", rec.RFtempM).
			Float64Column("lRtempM", rec.LRtempM).
			Float64Column("rRtempM", rec.RRtempM).
			At(ctx, ts)

		if err != nil {
			log.Printf("Failed to write telemetry record: %v", err)
			return err
		}
	}
	if err := sender.Flush(ctx); err != nil {
		log.Printf("Failed to flush telemetry batch: %v", err)
		return err
	}

	return nil
}
