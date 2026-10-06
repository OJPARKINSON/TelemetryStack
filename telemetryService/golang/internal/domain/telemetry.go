package domain

import (
	"time"

	"github.com/ojparkinson/telemetryService/internal/messaging"
)

type TelemetryPoint struct {
	// Identifiers
	SessionID   string `json:"session_id"`
	TrackName   string `json:"track_name"`
	TrackID     string `json:"track_id"`
	LapID       string `json:"lap_id"`
	SessionNum  string `json:"session_num"`
	SessionType string `json:"session_type"`
	SessionName string `json:"session_name"`
	CarID       string `json:"car_id"`

	// Timing
	SessionTime float64   `json:"session_time"`
	Timestamp   time.Time `json:"timestamp"`

	// Core telemetry
	Speed              float64 `json:"speed"`
	RPM                float64 `json:"rpm"`
	Throttle           float64 `json:"throttle"`
	Brake              float64 `json:"brake"`
	Gear               uint32  `json:"gear"`
	LapDistPct         float64 `json:"lap_dist_pct"`
	SteeringWheelAngle float64 `json:"steering_wheel_angle"`
	PlayerCarPosition  float64 `json:"player_car_position"`

	// Position
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
	Alt float64 `json:"alt"`

	// Velocity
	VelocityX float64 `json:"velocity_x"`
	VelocityY float64 `json:"velocity_y"`
	VelocityZ float64 `json:"velocity_z"`

	// Acceleration
	LatAccel  float64 `json:"lat_accel"`
	LongAccel float64 `json:"long_accel"`
	VertAccel float64 `json:"vert_accel"`

	// Orientation
	Pitch    float64 `json:"pitch"`
	Roll     float64 `json:"roll"`
	Yaw      float64 `json:"yaw"`
	YawNorth float64 `json:"yaw_north"`

	// Lap times
	LapCurrentLapTime float64 `json:"lap_current_lap_time"`
	LapLastLapTime    float64 `json:"lap_last_lap_time"`
	LapDeltaToBestLap float64 `json:"lap_delta_to_best_lap"`

	// Car state
	FuelLevel float64 `json:"fuel_level"`
	Voltage   float64 `json:"voltage"`
	WaterTemp float64 `json:"water_temp"`

	// Tyres
	LFpressure float64 `json:"lf_pressure"`
	RFpressure float64 `json:"rf_pressure"`
	LRpressure float64 `json:"lr_pressure"`
	RRpressure float64 `json:"rr_pressure"`
	LFtempM    float64 `json:"lf_temp_m"`
	RFtempM    float64 `json:"rf_temp_m"`
	LRtempM    float64 `json:"lr_temp_m"`
	RRtempM    float64 `json:"rr_temp_m"`
}

func TelemetryPointFromProto(rec *messaging.Telemetry, sessionId, CarId string) TelemetryPoint {
	timestamp := time.Now()
	if rec.TickTime != nil {
		timestamp = rec.TickTime.AsTime()
	}

	return TelemetryPoint{
		SessionID: sessionId,
		CarID:     CarId,

		TrackName:          rec.TrackName,
		TrackID:            rec.TrackId,
		LapID:              rec.LapId,
		SessionNum:         rec.SessionNum,
		SessionType:        rec.SessionType,
		SessionName:        rec.SessionName,
		SessionTime:        rec.SessionTime,
		Speed:              rec.Speed,
		RPM:                rec.Rpm,
		Throttle:           rec.Throttle,
		Brake:              rec.Brake,
		Gear:               rec.Gear,
		LapDistPct:         rec.LapDistPct,
		SteeringWheelAngle: rec.SteeringWheelAngle,
		PlayerCarPosition:  rec.PlayerCarPosition,
		Lat:                rec.Lat,
		Lon:                rec.Lon,
		Alt:                rec.Alt,
		VelocityX:          rec.VelocityX,
		VelocityY:          rec.VelocityY,
		VelocityZ:          rec.VelocityZ,
		LatAccel:           rec.LatAccel,
		LongAccel:          rec.LongAccel,
		VertAccel:          rec.VertAccel,
		Pitch:              rec.Pitch,
		Roll:               rec.Roll,
		Yaw:                rec.Yaw,
		YawNorth:           rec.YawNorth,
		LapCurrentLapTime:  rec.LapCurrentLapTime,
		LapLastLapTime:     rec.LapLastLapTime,
		LapDeltaToBestLap:  rec.LapDeltaToBestLap,
		FuelLevel:          rec.FuelLevel,
		Voltage:            rec.Voltage,
		WaterTemp:          rec.WaterTemp,
		LFpressure:         rec.LFpressure,
		RFpressure:         rec.RFpressure,
		LRpressure:         rec.LRpressure,
		RRpressure:         rec.RRpressure,
		LFtempM:            rec.LFtempM,
		RFtempM:            rec.RFtempM,
		LRtempM:            rec.LRtempM,
		RRtempM:            rec.RRtempM,
		Timestamp:          timestamp,
	}
}
