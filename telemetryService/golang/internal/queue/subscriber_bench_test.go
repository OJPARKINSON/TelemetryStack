package queue

import (
	"testing"
	"time"

	"github.com/ojparkinson/telemetryService/internal/messaging"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BenchmarkCollectValidRecords benchmarks the current production code
// This calls the actual collectValidRecords function from subscriber.go
// Run with: go test -bench=BenchmarkCollectValidRecords -benchmem -memprofile=mem.prof
func BenchmarkCollectValidRecords(b *testing.B) {
	benchmarks := []struct {
		name            string
		numBatches      int
		recordsPerBatch int
	}{
		{"Small_5batches_100records", 5, 100},
		{"Medium_10batches_500records", 10, 500},
		{"Large_20batches_1000records", 20, 1000},
		{"XLarge_20batches_5000records", 20, 5000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			items := generateBatchItems(bm.numBatches, bm.recordsPerBatch)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Calls the actual production function
				_ = CollectValidRecords(items)
			}
		})
	}
}

// BenchmarkIsValidRecord benchmarks the validation function
func BenchmarkIsValidRecord(b *testing.B) {
	validRecord := generateTelemetryRecord("session-123", "Spa-Francorchamps")
	invalidRecord := &messaging.Telemetry{}

	b.Run("ValidRecord", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = IsValidRecord(validRecord)
		}
	})

	b.Run("InvalidRecord", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = IsValidRecord(invalidRecord)
		}
	})
}

// Helper functions for generating test data
func generateBatchItems(numBatches, recordsPerBatch int) []batchItem {
	items := make([]batchItem, numBatches)
	for i := 0; i < numBatches; i++ {
		items[i] = batchItem{
			batch: &messaging.TelemetryBatch{
				SessionId: "test-session",
				BatchId:   "batch-" + string(rune(i)),
				Records:   generateTelemetryRecords(recordsPerBatch),
			},
			deliveryTag: uint64(i + 1),
		}
	}
	return items
}

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
		SessionId:          sessionID,
		TrackName:          trackName,
		TrackId:            "14",
		LapId:              "lap-1",
		SessionNum:         "0",
		SessionType:        "Race",
		SessionName:        "Feature Race",
		CarId:              "mercedes_amg_gt3",
		Speed:              150.5,
		LapDistPct:         0.45,
		SessionTime:        123.45,
		Lat:                50.4372,
		Lon:                5.9714,
		Gear:               4,
		PlayerCarPosition:  3,
		Throttle:           0.85,
		Brake:              0.0,
		SteeringWheelAngle: -0.15,
		Rpm:                7500.0,
		VelocityX:          25.5,
		VelocityY:          0.5,
		VelocityZ:          35.2,
		FuelLevel:          45.5,
		Alt:                123.4,
		LatAccel:           1.2,
		LongAccel:          0.8,
		VertAccel:          0.1,
		Pitch:              0.05,
		Roll:               -0.02,
		Yaw:                1.57,
		YawNorth:           3.14,
		Voltage:            13.8,
		WaterTemp:          85.5,
		LapCurrentLapTime:  95.234,
		LapLastLapTime:     94.567,
		LapDeltaToBestLap:  0.667,
		LFpressure:         28.5,
		RFpressure:         28.6,
		LRpressure:         27.8,
		RRpressure:         27.9,
		LFtempM:            85.2,
		RFtempM:            86.1,
		LRtempM:            84.5,
		RRtempM:            85.0,
		TickTime:           timestamppb.New(now),
	}
}
