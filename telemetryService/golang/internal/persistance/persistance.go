package persistance

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/ojparkinson/telemetryService/internal/messaging"
	qdb "github.com/questdb/go-questdb-client/v4"
)

func WriteBatch(sender qdb.LineSender, records []*messaging.Telemetry) error {
	ctx := context.Background()
	const flushInterval = 100000

	for i, record := range records {
		sender.Table("TelemetryTicks").
			Symbol("session_id", sanitise(record.SessionId)).
			Symbol("track_name", sanitise(record.TrackName)).
			Symbol("track_id", sanitise(record.TrackId)).
			Symbol("lap_id", sanitise(record.LapId)).
			Symbol("session_num", sanitise(record.SessionNum)).
			Symbol("session_type", sanitise(record.SessionType)).
			Symbol("session_name", sanitise(record.SessionName)).
			Symbol("car_id", sanitise(record.CarId)).
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
			At(ctx, tickTime(record))

		// Flush every 10K records to keep memory and network packets reasonable
		if (i+1)%flushInterval == 0 {
			if err := sender.Flush(ctx); err != nil {
				return fmt.Errorf("flush failed at record %d: %w", i, err)
			}
		}
	}

	// Final flush for any remaining records
	err := sender.Flush(ctx)
	if err != nil {
		return fmt.Errorf("final flush failed: %w", err)
	}

	// fmt.Printf("wrote %d records to QuestDb\n", len(records))
	return nil
}

func tickTime(record *messaging.Telemetry) time.Time {
	if record.TickTime != nil {
		return record.TickTime.AsTime()
	}
	log.Printf("WARNING: record has nil TickTime, using time.Now() as fallback")
	return time.Now()
}

func sanitise(value string) string {
	if value == "" {
		return "unknown"
	}

	// Fast path: check if sanitization needed
	needsSanitization := false
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c == ',' || c == ' ' || c == '=' || c == '\n' ||
			c == '\r' || c == '"' || c == '\'' || c == '\\' {
			needsSanitization = true
			break
		}
	}

	if !needsSanitization {
		trimmed := strings.TrimSpace(value)
		if trimmed == value {
			return value // Zero allocations
		}
		return trimmed
	}

	// Single-pass with pre-allocated builder
	var builder strings.Builder
	builder.Grow(len(value))

	for i := 0; i < len(value); i++ {
		c := value[i]
		switch c {
		case ',', ' ', '=', '\n', '\r', '"', '\'', '\\':
			builder.WriteByte('_')
		default:
			builder.WriteByte(c)
		}
	}

	return strings.TrimSpace(builder.String())
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

func mapToTelemetry(m map[string]interface{}) messaging.Telemetry {
	return messaging.Telemetry{
		LapId:       getString(m, "lap_id"),
		SessionId:   getString(m, "session_id"),
		SessionNum:  getString(m, "session_num"),
		SessionType: getString(m, "session_type"),
		SessionName: getString(m, "session_name"),
		CarId:       getString(m, "car_id"),
		TrackName:   getString(m, "track_name"),
		TrackId:     getString(m, "track_id"),

		Lat:        getFloat64(m, "lat"),
		Lon:        getFloat64(m, "lon"),
		Alt:        getFloat64(m, "alt"),
		LapDistPct: getFloat64(m, "lap_dist_pct"),

		Speed:     getFloat64(m, "speed"),
		VelocityX: getFloat64(m, "velocity_x"),
		VelocityY: getFloat64(m, "velocity_y"),
		VelocityZ: getFloat64(m, "velocity_z"),

		// Driver Inputs
		Throttle:           getFloat64(m, "throttle"),
		Brake:              getFloat64(m, "brake"),
		SteeringWheelAngle: getFloat64(m, "steering_wheel_angle"),
		Gear:               uint32(getInt(m, "gear")),

		// Engine
		Rpm:       getFloat64(m, "rpm"),
		FuelLevel: getFloat64(m, "fuel_level"),

		// Forces
		LatAccel:  getFloat64(m, "lat_accel"),
		LongAccel: getFloat64(m, "long_accel"),
		VertAccel: getFloat64(m, "vert_accel"),

		// Orientation
		Pitch:    getFloat64(m, "pitch"),
		Roll:     getFloat64(m, "roll"),
		Yaw:      getFloat64(m, "yaw"),
		YawNorth: getFloat64(m, "yaw_north"),

		// Telemetry
		Voltage:   getFloat64(m, "voltage"),
		WaterTemp: getFloat64(m, "water_temp"),

		// Tire Pressures
		LFpressure: getFloat64(m, "lFpressure"),
		RFpressure: getFloat64(m, "rFpressure"),
		LRpressure: getFloat64(m, "lRpressure"),
		RRpressure: getFloat64(m, "rRpressure"),

		// Tire Temps
		LFtempM: getFloat64(m, "lFtempM"),
		RFtempM: getFloat64(m, "rFtempM"),
		LRtempM: getFloat64(m, "lRtempM"),
		RRtempM: getFloat64(m, "rRtempM"),

		// Timing
		SessionTime:       getFloat64(m, "session_time"),
		LapCurrentLapTime: getFloat64(m, "lap_current_lap_time"),
		LapLastLapTime:    getFloat64(m, "lapLastLapTime"),
		LapDeltaToBestLap: getFloat64(m, "lapDeltaToBestLap"),
		PlayerCarPosition: uint32(getInt(m, "player_car_position")),

		// WorkerId not in DB - will be 0
		// TickTime not mapped - use raw timestamp if needed
	}
}

// Keep helper functions from before
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok && v != nil {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return 0.0
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok && v != nil {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		}
	}
	return 0
}
