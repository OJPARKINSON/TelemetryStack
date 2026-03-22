package domain

import (
	"context"

	"github.com/ojparkinson/telemetryService/internal/messaging"
)

type SessionRepository interface {
	ListSessions(ctx context.Context) ([]Session, error)
	ListLaps(ctx context.Context, sessionId string) ([]Lap, error)
	GetLapTelemetry(ctx context.Context, sessionID, lapId string) ([]TelemetryPoint, error)
}

type TelemetryWriter interface {
	WriteBatch(ctx context.Context, records []*messaging.Telemetry) error
}
