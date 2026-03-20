package messaging

import (
	"sync"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type TelemetryRecord struct {
	CarID              string  `json:"car_id"`
	Brake              float32 `json:"brake"`
	FuelLevel          float32 `json:"fuel_level"`
	Gear               int16   `json:"gear"`
	TrackName          string  `json:"track_name"`
	TrackID            int32   `json:"track_id"`
	LapCurrentLapTime  float32 `json:"lap_current_lap_time"`
	LapDistPct         float32 `json:"lap_dist_pct"`
	LapID              string  `json:"lap_id"`
	Lat                float32 `json:"lat"`
	Lon                float32 `json:"lon"`
	PlayerCarPosition  int16   `json:"player_car_position"`
	RPM                float32 `json:"rpm"`
	SessionID          string  `json:"session_id"`
	SessionNum         string  `json:"session_num"`
	SessionType        string  `json:"session_type"`
	SessionName        string  `json:"session_name"`
	SessionTime        float32 `json:"session_time"`
	Speed              float32 `json:"speed"`
	SteeringWheelAngle float32 `json:"steering_wheel_angle"`
	Throttle           float32 `json:"throttle"`
	TickTime           string  `json:"tick_time"`
	VelocityX          float32 `json:"velocity_x"`
	VelocityY          float32 `json:"velocity_y"`
	VelocityZ          float32 `json:"velocity_z"`
	Alt                float32 `json:"alt"`
	LatAccel           float32 `json:"lat_accel"`
	LongAccel          float32 `json:"long_accel"`
	VertAccel          float32 `json:"vert_accel"`
	Pitch              float32 `json:"pitch"`
	Roll               float32 `json:"roll"`
	Yaw                float32 `json:"yaw"`
	YawNorth           float32 `json:"yaw_north"`
	Voltage            float32 `json:"voltage"`
	LapLastLapTime     float32 `json:"lapLastLapTime"`
	WaterTemp          float32 `json:"waterTemp"`
	LapDeltaToBestLap  float32 `json:"lapDeltaToBestLap"`
	LFPressure         float32 `json:"lFpressure"`
	RFPressure         float32 `json:"rFpressure"`
	LRPressure         float32 `json:"lRpressure"`
	RRPressure         float32 `json:"rRpressure"`
	LFTempM            float32 `json:"lFtempM"`
	RFTempM            float32 `json:"rFtempM"`
	LRTempM            float32 `json:"lRtempM"`
	RRTempM            float32 `json:"rRtempM"`

	GroupNum int `json:"-"`
	WorkerID int `json:"-"`
}

func (tr *TelemetryRecord) Reset() {
	*tr = TelemetryRecord{}
}

type RecordPool struct {
	pool sync.Pool
}

func NewRecordPool() *RecordPool {
	return &RecordPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &TelemetryRecord{}
			},
		},
	}
}

func (rp *RecordPool) Get() *TelemetryRecord {
	return rp.pool.Get().(*TelemetryRecord)
}

func (rp *RecordPool) Put(record *TelemetryRecord) {
	record.Reset()
	rp.pool.Put(record)
}

type BatchPool struct {
	pool sync.Pool
}

func NewBatchPool(batchSize int) *BatchPool {
	return &BatchPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &TelemetryBatch{
					Records: make([]*Telemetry, 0, batchSize),
				}
			},
		},
	}
}

func (bp *BatchPool) Get() *TelemetryBatch {
	return bp.pool.Get().(*TelemetryBatch)
}

func (bp *BatchPool) Put(batch *TelemetryBatch) {
	batch.Records = batch.Records[:0]
	batch.BatchId = ""
	batch.Timestamp = &timestamppb.Timestamp{}
	batch.WorkerId = 0
	batch.SessionId = ""
	bp.pool.Put(batch)
}
