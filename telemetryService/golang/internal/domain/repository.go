package domain

import (
	"context"
	"errors"
)

var ErrQueueFull = errors.New("ingest queue full")
var ErrNotFound = errors.New("not found")

type SessionRepository interface {
	ListSessions(ctx context.Context) ([]Session, error)
	ListLaps(ctx context.Context, sessionId string) ([]Lap, error)
	GetLapTelemetry(ctx context.Context, sessionID, lapId string) ([]TelemetryPoint, error)
	UpsertTrack(ctx context.Context, track Track) error
	GetTrack(ctx context.Context, trackID string) (Track, error)
	GetLapTrackID(ctx context.Context, sessionID, lapID string) (string, error)
}

type TelemetryWriter interface {
	WriteBatch(ctx context.Context, records []*TelemetryPoint) error
}
