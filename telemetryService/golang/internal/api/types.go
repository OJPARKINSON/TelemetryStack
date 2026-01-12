package api

import "time"

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
