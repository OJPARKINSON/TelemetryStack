package messaging

import (
	"fmt"
	"strconv"
	"time"

	"github.com/OJPARKINSON/ibt"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TransformStructBatch(ticks []*ibt.TelemetryTick) ([]*Telemetry, error) {
	result := make([]*Telemetry, len(ticks))

	for i, tick := range ticks {
		result[i] = &Telemetry{
			LapId:              strconv.Itoa(int(tick.LapID)),
			Speed:              tick.Speed,
			LapDistPct:         tick.LapDistPct,
			Throttle:           tick.Throttle,
			Brake:              tick.Brake,
			Gear:               tick.Gear,
			Rpm:                tick.RPM,
			SteeringWheelAngle: tick.SteeringWheelAngle,
			VelocityX:          tick.VelocityX,
			VelocityY:          tick.VelocityY,
			VelocityZ:          tick.VelocityZ,
			Lat:                tick.Lat,
			Lon:                tick.Lon,
			SessionTime:        tick.SessionTime,
			PlayerCarPosition:  tick.PlayerCarPosition,
			FuelLevel:          tick.FuelLevel,
			CarId:              strconv.Itoa(int(tick.PlayerCarIdx)),
			SessionNum:         strconv.Itoa(int(tick.SessionNum)),
			Alt:                tick.Alt,
			LatAccel:           tick.LatAccel,
			LongAccel:          tick.LongAccel,
			VertAccel:          tick.VertAccel,
			Pitch:              tick.Pitch,
			Roll:               tick.Roll,
			Yaw:                tick.Yaw,
			YawNorth:           tick.YawNorth,
			Voltage:            tick.Voltage,
			LapLastLapTime:     tick.LapLastLapTime,
			WaterTemp:          tick.WaterTemp,
			LapDeltaToBestLap:  tick.LapDeltaToBestLap,
			LapCurrentLapTime:  tick.LapCurrentLapTime,
			LFpressure:         tick.LFpressure,
			RFpressure:         tick.RFpressure,
			LRpressure:         tick.LRpressure,
			RRpressure:         tick.RRpressure,
			LFtempM:            tick.LFtempM,
			RFtempM:            tick.RFtempM,
			LRtempM:            tick.LRtempM,
			RRtempM:            tick.RRtempM,

			// Session metadata
			SessionId:   tick.SessionID,
			SessionType: tick.SessionType,
			SessionName: tick.SessionName,
			TrackName:   tick.TrackName,
			TrackId:     fmt.Sprintf("%d", tick.TrackID),
			WorkerId:    uint32(tick.WorkerID),

			TickTime: timestamppb.New(tick.TickTime),
		}
	}

	return result, nil
}

func (ps *PubSub) ExecStructs(ticks []*ibt.TelemetryTick) error {
	if len(ticks) == 0 {
		return nil
	}

	// Extract workerID from first tick if available
	if len(ticks) > 0 {
		ps.workerID = ticks[0].WorkerID
	}

	// Transform batch (no lock needed - pure transformation)
	protoTicks, err := TransformStructBatch(ticks)
	if err != nil {
		return fmt.Errorf("failed to transform struct batch: %w", err)
	}

	// Pre-calculate batch size ONCE instead of per-tick
	// Use fixed estimate to avoid expensive proto.Size() calls
	const estimatedTickSize = 512
	batchSize := int64(len(protoTicks) * estimatedTickSize)

	// Lock ONCE for the entire batch operation
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Check if adding this batch would exceed limits - flush first if needed
	wouldExceedRecords := len(ps.recordBatch)+len(protoTicks) >= ps.batchSizeRecords
	wouldExceedBytes := ps.totalBytes+batchSize >= int64(ps.batchSizeBytes)
	shouldFlushTime := time.Since(ps.lastFlush) > time.Duration(ps.config.BatchTimeout)

	if (wouldExceedRecords || wouldExceedBytes || shouldFlushTime) && len(ps.recordBatch) > 0 {
		if err := ps.flushBatchInternal(); err != nil {
			return err
		}
	}

	// Append entire batch at once (fast slice append)
	ps.recordBatch = append(ps.recordBatch, protoTicks...)
	ps.totalRecords += len(protoTicks)
	ps.totalBytes += batchSize

	return nil
}
