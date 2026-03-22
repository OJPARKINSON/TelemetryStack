package domain

type TelemetryPoint struct {
	SessionTime float64 `json:"session_time"`

	Speed              float64 `json:"speed"`                // km/h
	RPM                float64 `json:"rpm"`
	Throttle           float64 `json:"throttle"`             // 0-100%
	Brake              float64 `json:"brake"`                // 0-100%
	Gear               uint32  `json:"gear"`
	LapDistPct         float64 `json:"lap_dist_pct"`         // 0-100%
	SteeringWheelAngle float64 `json:"steering_wheel_angle"` // degrees

	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
	Alt float64 `json:"alt"`

	VelocityX float64 `json:"velocity_x"`
	VelocityY float64 `json:"velocity_y"`
	VelocityZ float64 `json:"velocity_z"`

	LatAccel  float64 `json:"lat_accel"`
	LongAccel float64 `json:"long_accel"`
	VertAccel float64 `json:"vert_accel"`

	Pitch    float64 `json:"pitch"`
	Roll     float64 `json:"roll"`
	Yaw      float64 `json:"yaw"`
	YawNorth float64 `json:"yaw_north"`

	FuelLevel         float64 `json:"fuel_level"`
	LapCurrentLapTime float64 `json:"lap_current_lap_time"`
	PlayerCarPosition uint32  `json:"player_car_position"`
	TrackName         string  `json:"track_name"`
	SessionNum        string  `json:"session_num"`
}
