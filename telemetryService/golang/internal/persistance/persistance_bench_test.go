package persistance

import (
	"context"
	"testing"
	"time"

	"github.com/ojparkinson/telemetryService/internal/messaging"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BenchmarkSanitise benchmarks the string sanitization function
func BenchmarkSanitise(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{"Empty", ""},
		{"Clean", "clean_string_no_special_chars"},
		{"WithSpaces", "track name with spaces"},
		{"WithCommas", "value,with,commas"},
		{"WithQuotes", `value"with"quotes`},
		{"AllSpecialChars", `track,name=value with "quotes" and\backslash`},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = sanitise(tc.input)
			}
		})
	}
}

// BenchmarkValidateDouble benchmarks float64 validation
func BenchmarkValidateDouble(b *testing.B) {
	testCases := []struct {
		name  string
		value float64
	}{
		{"Normal", 123.456},
		{"Zero", 0.0},
		{"Negative", -123.456},
		{"Large", 1e10},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = validateDouble(tc.value)
			}
		})
	}
}

// BenchmarkValidateInt benchmarks uint32 validation
func BenchmarkValidateInt(b *testing.B) {
	testCases := []struct {
		name  string
		value uint32
	}{
		{"Normal", 42},
		{"Zero", 0},
		{"Max", 0xFFFFFFFF},
		{"Large", 1000000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = validateInt(tc.value)
			}
		})
	}
}

// MockLineSender implements a no-op version of qdb.LineSender for benchmarking
type MockLineSender struct {
	rowCount int
}

func (m *MockLineSender) Table(name string) *MockLineSender {
	return m
}

func (m *MockLineSender) Symbol(name, value string) *MockLineSender {
	return m
}

func (m *MockLineSender) StringColumn(name, value string) *MockLineSender {
	return m
}

func (m *MockLineSender) Int64Column(name string, value int64) *MockLineSender {
	return m
}

func (m *MockLineSender) Float64Column(name string, value float64) *MockLineSender {
	return m
}

func (m *MockLineSender) At(ctx context.Context, t time.Time) *MockLineSender {
	m.rowCount++
	return m
}

func (m *MockLineSender) Flush(ctx context.Context) error {
	m.rowCount = 0
	return nil
}

func (m *MockLineSender) Close(ctx context.Context) error {
	return nil
}

// BenchmarkRecordSerialization benchmarks the conversion of a single telemetry record
func BenchmarkRecordSerialization(b *testing.B) {
	record := generateTelemetryRecord("session-123", "Spa-Francorchamps")
	sender := &MockLineSender{}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
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
}

// BenchmarkBatchSerialization benchmarks processing different batch sizes
func BenchmarkBatchSerialization(b *testing.B) {
	benchmarks := []struct {
		name  string
		count int
	}{
		{"100records", 100},
		{"1000records", 1000},
		{"5000records", 5000},
		{"10000records", 10000},
		{"25000records", 25000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			records := generateTelemetryRecords(bm.count)
			sender := &MockLineSender{}
			ctx := context.Background()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
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
						At(ctx, record.TickTime.AsTime())
				}
				sender.Flush(ctx)
			}
		})
	}
}

// Helper functions
func generateTelemetryRecords(count int) []*messaging.Telemetry {
	records := make([]*messaging.Telemetry, count)
	for i := 0; i < count; i++ {
		records[i] = generateTelemetryRecord("session-123", "Spa-Francorchamps")
	}
	return records
}

func generateTelemetryRecord(sessionID, trackName string) *messaging.Telemetry {
	now := time.Now()
	return &messaging.Telemetry{
		SessionId:         sessionID,
		TrackName:         trackName,
		TrackId:           "14",
		LapId:             "lap-1",
		SessionNum:        "0",
		SessionType:       "Race",
		SessionName:       "Feature Race",
		CarId:             "mercedes_amg_gt3",
		Speed:             150.5,
		LapDistPct:        0.45,
		SessionTime:       123.45,
		Lat:               50.4372,
		Lon:               5.9714,
		Gear:              4,
		PlayerCarPosition: 3,
		Throttle:          0.85,
		Brake:             0.0,
		SteeringWheelAngle: -0.15,
		Rpm:               7500.0,
		VelocityX:         25.5,
		VelocityY:         0.5,
		VelocityZ:         35.2,
		FuelLevel:         45.5,
		Alt:               123.4,
		LatAccel:          1.2,
		LongAccel:         0.8,
		VertAccel:         0.1,
		Pitch:             0.05,
		Roll:              -0.02,
		Yaw:               1.57,
		YawNorth:          3.14,
		Voltage:           13.8,
		WaterTemp:         85.5,
		LapCurrentLapTime: 95.234,
		LapLastLapTime:    94.567,
		LapDeltaToBestLap: 0.667,
		LFpressure:        28.5,
		RFpressure:        28.6,
		LRpressure:        27.8,
		RRpressure:        27.9,
		LFtempM:           85.2,
		RFtempM:           86.1,
		LRtempM:           84.5,
		RRtempM:           85.0,
		TickTime:          timestamppb.New(now),
	}
}
