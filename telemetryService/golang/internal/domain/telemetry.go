package domain

type TelemetryPoint struct {
	Index       int     `json:"index"`
	SessionTime float64 `json:"sessionTime"`

	Speed              float64 `json:"Speed"` // km/h
	RPM                float64 `json:"RPM"`
	Throttle           float64 `json:"Throttle"` // 0-100%
	Brake              float64 `json:"Brake"`    // 0-100%
	Gear               uint32  `json:"Gear"`
	LapDistPct         float64 `json:"LapDistPct"`         // 0-100%
	SteeringWheelAngle float64 `json:"SteeringWheelAngle"` // degrees

	Lat float64 `json:"Lat"`
	Lon float64 `json:"Lon"`
	Alt float64 `json:"Alt"`

	VelocityX float64 `json:"VelocityX"`
	VelocityY float64 `json:"VelocityY"`
	VelocityZ float64 `json:"VelocityZ"`

	LatAccel  float64 `json:"LatAccel"`
	LongAccel float64 `json:"LongAccel"`
	VertAccel float64 `json:"VertAccel"`

	Pitch    float64 `json:"Pitch"`
	Roll     float64 `json:"Roll"`
	Yaw      float64 `json:"Yaw"`
	YawNorth float64 `json:"YawNorth"`

	FuelLevel         float64 `json:"FuelLevel"`
	LapCurrentLapTime float64 `json:"LapCurrentLapTime"`
	PlayerCarPosition uint32  `json:"PlayerCarPosition"`
	TrackName         string  `json:"TrackName"`
	SessionNum        string  `json:"SessionNum"`
}
