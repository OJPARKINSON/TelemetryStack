package questdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ojparkinson/telemetryService/internal/db/generated"
	"github.com/ojparkinson/telemetryService/internal/domain"
	"github.com/ojparkinson/telemetryService/internal/metrics"
)

func observeQuery(operation string, start time.Time, err error) {
	metrics.QuestDBQueryDuration.WithLabelValues(operation).Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.QuestDBQueryErrors.WithLabelValues(operation).Inc()
	}
}

func (r *Repository) ListSessions(ctx context.Context) (_ []domain.Session, err error) {
	start := time.Now()
	defer func() { observeQuery("list_sessions", start, err) }()

	rows, err := r.queries.ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	sessions := make([]domain.Session, len(rows))
	for i, row := range rows {

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
			MaxLapID:    int(row.MaxLapID),
			LastUpdated: lastUpdated,
		}
	}
	return sessions, nil
}

func (r *Repository) ListLaps(ctx context.Context, sessionsID string) (_ []domain.Lap, err error) {
	start := time.Now()
	defer func() { observeQuery("list_laps", start, err) }()

	rows, err := r.queries.ListLaps(ctx, pgtype.Text{String: sessionsID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list laps: %w", err)
	}

	laps := make([]domain.Lap, len(rows))
	for i, row := range rows {
		laps[i] = domain.Lap{LapID: int(row)}
	}
	return laps, nil
}

func (r *Repository) GetLapTelemetry(ctx context.Context, sessionID, lapID string) (_ []domain.TelemetryPoint, err error) {
	start := time.Now()
	defer func() { observeQuery("get_lap_telemetry", start, err) }()

	lapIDInt, err := strconv.ParseInt(lapID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid lap_id: %w", err)
	}
	rows, err := r.queries.GetLapTelemetry(ctx, generated.GetLapTelemetryParams{
		SessionID: pgtype.Text{String: sessionID, Valid: true},
		LapID:     pgtype.Int4{Int32: int32(lapIDInt), Valid: true},
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

func telemetryPointFromRow(row generated.GetLapTelemetryRow) domain.TelemetryPoint {
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
		VelocityX:          row.VelocityX.Float64,
		VelocityY:          row.VelocityY.Float64,
		VelocityZ:          row.VelocityZ.Float64,
		LatAccel:           row.LatAccel.Float64,
		FuelLevel:          row.FuelLevel.Float64,
		LapCurrentLapTime:  row.LapCurrentLapTime.Float64,
		PlayerCarPosition:  float64(row.PlayerCarPosition.Int32),
		TrackName:          row.TrackName.String,
		SessionNum:         row.SessionNum.String,
		SessionTime:        row.SessionTime.Float64,
	}
}

func (r *Repository) UpsertTrack(ctx context.Context, track domain.Track) (err error) {
	start := time.Now()
	defer func() { observeQuery("upsert_track", start, err) }()

	sectors, err := json.Marshal(track.SectorStartPct)
	if err != nil {
		return fmt.Errorf("marshal sectors: %w", err)
	}
	err = r.queries.UpsertTrack(ctx, generated.UpsertTrackParams{
		TrackID:        pgtype.Text{String: track.TrackID, Valid: true},
		TrackName:      pgtype.Text{String: track.TrackName, Valid: true},
		SectorStartPct: pgtype.Text{String: string(sectors), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("upsert track: %w", err)
	}
	return nil
}

func (r *Repository) GetTrack(ctx context.Context, trackID string) (_ domain.Track, err error) {
	start := time.Now()
	defer func() { observeQuery("get_track", start, err) }()

	row, err := r.queries.GetTrack(ctx, pgtype.Text{String: trackID, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Track{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Track{}, fmt.Errorf("get track: %w", err)
	}

	track := domain.Track{TrackID: row.TrackID.String, TrackName: row.TrackName.String}
	if err := json.Unmarshal([]byte(row.SectorStartPct.String), &track.SectorStartPct); err != nil {
		return domain.Track{}, fmt.Errorf("unmarshal sectors: %w", err)
	}
	return track, nil
}

func (r *Repository) GetLapTrackID(ctx context.Context, sessionID, lapID string) (_ string, err error) {
	start := time.Now()
	defer func() { observeQuery("get_lap_track_id", start, err) }()

	lapIDInt, err := strconv.ParseInt(lapID, 10, 32)
	if err != nil {
		return "", fmt.Errorf("invalid lap_id: %w", err)
	}
	trackID, err := r.queries.GetLapTrackID(ctx, generated.GetLapTrackIDParams{
		SessionID: pgtype.Text{String: sessionID, Valid: true},
		LapID:     pgtype.Int4{Int32: int32(lapIDInt), Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get lap track id: %w", err)
	}
	return trackID.String, nil
}
