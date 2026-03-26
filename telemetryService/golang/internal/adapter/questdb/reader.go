package questdb

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ojparkinson/telemetryService/internal/db/generated"
	"github.com/ojparkinson/telemetryService/internal/domain"
)

func (r *Repository) ListSessions(ctx context.Context) ([]domain.Session, error) {
	rows, err := r.queries.ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	sessions := make([]domain.Session, len(rows))
	for i, row := range rows {
		var maxLapID int
		if v, ok := row.MaxLapID.(int); ok {
			maxLapID = v
		} else if v, ok := row.MaxLapID.(int); ok {
			maxLapID = v
		}

		var lastUpdated time.Time
		if v, ok := row.LastUpdated.(time.Time); ok {
			lastUpdated = v
		} else if v, ok := row.LastUpdated.(pgtype.Timestamp); ok && v.Valid {
			lastUpdated = v.Time
		}

		sessions[i] = domain.Session{
			SessionID:   row.SessionID.String,
			TrackName:   row.TrackName.String,
			SessionName: row.SessionName.String,
			MaxLapID:    maxLapID,
			LastUpdated: lastUpdated,
		}
	}
	return sessions, nil
}

func (r *Repository) ListLaps(ctx context.Context, sessionsID string) ([]domain.Lap, error) {
	rows, err := r.queries.ListLaps(ctx, pgtype.Text{String: sessionsID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list laps: %w", err)
	}

	laps := make([]domain.Lap, len(rows))
	for i, row := range rows {
		lapID, _ := strconv.Atoi(row.String)
		laps[i] = domain.Lap{LapID: lapID}
	}
	return laps, nil
}

func (r *Repository) GetLapTelemetry(ctx context.Context, sessionID, lapID string) ([]domain.TelemetryPoint, error) {
	rows, err := r.queries.GetLapTelemetry(ctx, generated.GetLapTelemetryParams{
		SessionID: pgtype.Text{String: sessionID, Valid: true},
		LapID:     pgtype.Text{String: lapID, Valid: true},
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
		Speed:              row.Speed.Float64,
		Throttle:           row.Throttle.Float64,
		Brake:              row.Brake.Float64,
		RPM:                row.Rpm.Float64,
		Gear:               uint32(row.Gear.Int32),
		LapDistPct:         row.LapDistPct.Float64,
		SteeringWheelAngle: row.SteeringWheelAngle.Float64,
		Lat:                row.Lat.Float64,
		Lon:                row.Lon.Float64,
		Alt:                row.Alt.Float64,
		VelocityX:          row.VelocityX.Float64,
		VelocityY:          row.VelocityY.Float64,
		VelocityZ:          row.VelocityZ.Float64,
		LatAccel:           row.LatAccel.Float64,
		LongAccel:          row.LongAccel.Float64,
		VertAccel:          row.VertAccel.Float64,
		Pitch:              row.Pitch.Float64,
		Roll:               row.Roll.Float64,
		Yaw:                row.Yaw.Float64,
		YawNorth:           row.YawNorth.Float64,
		FuelLevel:          row.FuelLevel.Float64,
		LapCurrentLapTime:  row.LapCurrentLapTime.Float64,
		PlayerCarPosition:  uint32(row.PlayerCarPosition.Int32),
		TrackName:          row.TrackName.String,
		SessionNum:         row.SessionNum.String,
		SessionTime:        row.SessionTime.Float64,
	}
}
