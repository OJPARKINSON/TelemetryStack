package domain

import "time"

type Session struct {
	SessionID   string    `json:"session_id"`
	TrackName   string    `json:"track_name"`
	SessionName string    `json:"session_name"`
	MaxLapID    int       `json:"max_lap_id"`
	LastUpdated time.Time `json:"last_updated"`
}

type Lap struct {
	LapID int `json:"lap_id"`
}

type Track struct {
	TrackID        string    `json:"track_id"`
	TrackName      string    `json:"track_name"`
	SectorStartPct []float64 `json:"sector_start_pct"` // fractions 0..1
}
