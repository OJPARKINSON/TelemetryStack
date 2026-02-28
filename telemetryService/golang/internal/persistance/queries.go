package persistance

import (
	"context"
	"fmt"

	"github.com/ojparkinson/telemetryService/internal/config"
	"github.com/ojparkinson/telemetryService/internal/messaging"
)

type QueryExecutor struct {
	Config *config.Config
}

func (s *QueryExecutor) QuerySession(ctx context.Context, sessionID string) ([]map[string]interface{}, error) {
	query := `
		SELECT DISTINCT session_id, track_name, session_name,
					MAX(lap_id) as max_lap_id,
					MAX(timestamp) as last_updated
		FROM TelemetryTicks
		WHERE session_name = 'RACE' AND lap_id > 0
		GROUP BY session_id, track_name, session_name
		ORDER BY last_updated DESC
	`
	return ExecuteSelectQuery(query, s.Config)
}

func (s *QueryExecutor) QuerySessions(ctx context.Context) ([]map[string]interface{}, error) {
	query := `
		SELECT DISTINCT session_id, track_name, session_name,
                MAX(lap_id) as max_lap_id,
                MAX(timestamp) as last_updated
		FROM TelemetryTicks
		WHERE session_name = 'RACE' AND lap_id > 0
		GROUP BY session_id, track_name, session_name
		ORDER BY last_updated DESC
	`
	return ExecuteSelectQuery(query, s.Config)
}

func (s *QueryExecutor) QueryLaps(ctx context.Context, sessionID string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT lap_id 
		FROM TelemetryTicks 
		WHERE session_id = %s
		ORDER BY lap_id ASC
	`, sessionID)

	return ExecuteSelectQuery(query, s.Config)
}

func (s *QueryExecutor) QueryLap(ctx context.Context, sessionID string, lapID string) ([]messaging.Telemetry, error) {
	query := fmt.Sprintf(`
		SELECT * FROM TelemetryTicks
		WHERE session_name = 'RACE' AND session_id = '%s' AND lap_id = '%s'
		ORDER BY timestamp ASC
	`, sessionID, lapID)

	rows, err := ExecuteSelectQuery(query, s.Config)
	if err != nil {
		return nil, err
	}

	points := make([]messaging.Telemetry, len(rows))
	for i, row := range rows {
		points[i] = mapToTelemetry(row)
	}

	return points, nil
}

func (s *QueryExecutor) QueryGeneralLap(ctx context.Context, sessionID string, lapID string) ([]messaging.Telemetry, error) {
	query := fmt.Sprintf(`
		SELECT * FROM TelemetryTicks 
        WHERE session_id = %s 
        WHERE lap_id = %s 
        ORDER BY timestamp ASC
	`, sessionID, lapID)

	rows, err := ExecuteSelectQuery(query, s.Config)
	if err != nil {
		return nil, err
	}

	points := make([]messaging.Telemetry, len(rows))
	for i, row := range rows {
		points[i] = mapToTelemetry(row)
	}

	return points, nil
}
