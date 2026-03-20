package publisher

import (
	"fmt"
	"math/rand"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func GenerateBatch(numBatches, recordsPerBatch int) []*TelemetryBatch {
	items := make([]*TelemetryBatch, numBatches)
	for i := 0; i < numBatches; i++ {
		items[i] = &TelemetryBatch{
			SessionId: "test-session",
			BatchId:   "batch-" + string(rune(i)),
			Records:   GenerateRecords(recordsPerBatch),
		}
	}
	return items
}

func GenerateRecords(count int) []*Telemetry {
	now := time.Now()
	records := make([]*Telemetry, count)
	for i := 0; i < count; i++ {
		records[i] = &Telemetry{
			SessionId:          fmt.Sprintf("session-%d", rand.Int()),
			TrackName:          "Spa-Francorchamps",
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
	return records
}
