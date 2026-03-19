package questdb

import (
	"context"
	"fmt"

	"github.com/ojparkinson/telemetryService/internal/db/generated"
	"github.com/ojparkinson/telemetryService/internal/domain"
)

func (r *Repository) ListSessions(ctx context.Context) ([]domain.Session, error) {
	rows, err := r.queries.ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %W", err)
	}

	sessions := make([]domain.Session, len(rows))
	for i, row := range rows {
		sessions[i] = domain.Session{
			SessionID:   row.SessionID,
			TrackName:   row.TrackName,
			SessionName: row.SessionName,
			MaxLapID:    row.MaxLapID,
			LastUpdated: row.LastUpdated,
		}
	}
	return sessions, nil
}

func (r *Repository) ListLaps(ctx context.Context, sessionsID string) ([]domain.Lap, error) {
	rows, err := r.queries.ListLaps(ctx, sessionsID)
	if err != nil {
		return nil, fmt.Errorf("list laps: %w", err)
	}

	laps := make([]domain.Lap, len(rows))
	for i, row := range rows {
		laps[i] = domain.Lap{LapID: row.LapID}
	}
	return laps, nil
}

func (r *Repository) GetLapTelemetry(ctx context.Context, sessionID, lapID string) ([]domain.TelemetryPoint, error) {
	rows, err := r.queries.GetLapTelemetry(ctx, generated.GetLapTelemetryParams{
		SessionID: sessionID,
		LapID:     lapID,
	})
	if err != nil {
		return nil, fmt.Errorf("get lap telemetry: %w", err)
	}

	points := make([]domain.TelemetryPoint, len(rows))
	for i, row := range rows {
		points[i] = telemetryPointFromRow(row)
	}
	return points, nil
}

func telemetryPointFromRow(row generated.Telemetrytick) domain.TelemetryPoint {
	return domain.TelemetryPoint{
		Speed:    row.Speed,
		Throttle: row.Throttle,
		Brake:    row.Brake,
		RPM:      row.Rpm,
		Gear:     row.Gear,
		// ... all fields, but now it's typed — no getString/getFloat64 helpers
	}
}
