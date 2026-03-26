package domain

import (
	"context"
)

type SessionRepository interface {
	ListSessions(ctx context.Context) ([]Session, error)
	ListLaps(ctx context.Context, sessionId string) ([]Lap, error)
	GetLapTelemetry(ctx context.Context, sessionID, lapId string) ([]TelemetryPoint, error)
}

type TelemetryWriter interface {
	WriteBatch(ctx context.Context, records []*TelemetryPoint) error
}
