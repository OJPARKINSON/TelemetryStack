package api

import (
	"time"

	"github.com/ojparkinson/telemetryService/internal/messaging"
)

type Session struct {
	SessionID   string    `json:"session_id"`
	TrackName   string    `json:"track_name"`
	SessionName string    `json:"session_name"`
	MaxLapID    int       `json:"max_lap_id"`
	LastUpdated time.Time `json:"last_updated"`
}

type Lap struct {
	LapID string `json:"lap_id"`
}

type TelemetryDataPoint struct {
	Index       int     `json:"index"`
	SessionTime float64 `json:"sessionTime"`

	// Converted values (display-ready)
	Speed              float64 `json:"Speed"` // km/h
	RPM                float64 `json:"RPM"`
	Throttle           float64 `json:"Throttle"` // 0-100%
	Brake              float64 `json:"Brake"`    // 0-100%
	Gear               uint32  `json:"Gear"`
	LapDistPct         float64 `json:"LapDistPct"`         // 0-100%
	SteeringWheelAngle float64 `json:"SteeringWheelAngle"` // degrees

	// Position
	Lat float64 `json:"Lat"`
	Lon float64 `json:"Lon"`
	Alt float64 `json:"Alt"`

	// Motion
	VelocityX float64 `json:"VelocityX"`
	VelocityY float64 `json:"VelocityY"`
	VelocityZ float64 `json:"VelocityZ"`

	// Forces
	LatAccel  float64 `json:"LatAccel"`
	LongAccel float64 `json:"LongAccel"`
	VertAccel float64 `json:"VertAccel"`

	// Orientation
	Pitch    float64 `json:"Pitch"`
	Roll     float64 `json:"Roll"`
	Yaw      float64 `json:"Yaw"`
	YawNorth float64 `json:"YawNorth"`

	// Other
	FuelLevel         float64 `json:"FuelLevel"`
	LapCurrentLapTime float64 `json:"LapCurrentLapTime"`
	PlayerCarPosition uint32  `json:"PlayerCarPosition"`
	TrackName         string  `json:"TrackName"`
	SessionNum        string  `json:"SessionNum"`
}

func ConvertToDisplayFormat(raw []messaging.Telemetry) []TelemetryDataPoint {
	result := make([]TelemetryDataPoint, len(raw))

	for i, d := range raw {
		result[i] = TelemetryDataPoint{
			Index:       i,
			SessionTime: d.SessionTime,

			// Unit conversions matching frontend expectations
			Speed:              d.Speed * 3.6, // m/s → km/h
			RPM:                d.Rpm,
			Throttle:           d.Throttle * 100, // 0-1 → 0-100%
			Brake:              d.Brake * 100,    // 0-1 → 0-100%
			Gear:               d.Gear,
			LapDistPct:         d.LapDistPct * 100, // 0-1 → 0-100%
			SteeringWheelAngle: d.SteeringWheelAngle,

			// Position (no conversion)
			Lat: d.Lat,
			Lon: d.Lon,
			Alt: d.Alt,

			// Motion (no conversion)
			VelocityX: d.VelocityX,
			VelocityY: d.VelocityY,
			VelocityZ: d.VelocityZ,

			// Forces (no conversion)
			LatAccel:  d.LatAccel,
			LongAccel: d.LongAccel,
			VertAccel: d.VertAccel,

			// Orientation (no conversion)
			Pitch:    d.Pitch,
			Roll:     d.Roll,
			Yaw:      d.Yaw,
			YawNorth: d.YawNorth,

			// Other
			FuelLevel:         d.FuelLevel,
			LapCurrentLapTime: d.LapCurrentLapTime,
			PlayerCarPosition: d.PlayerCarPosition,
			TrackName:         d.TrackName,
			SessionNum:        d.SessionNum,
		}
	}

	return result
}
