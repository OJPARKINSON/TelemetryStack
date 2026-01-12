package persistance

import (
	"context"
	"fmt"
)

type QueryExecutor struct {
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
	return ExecuteSelectQuery(query)
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
	return ExecuteSelectQuery(query)
}

func (s *QueryExecutor) QueryLaps(ctx context.Context, sessionID string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT lap_id 
		FROM TelemetryTicks 
		WHERE session_id = %s
		ORDER BY lap_id ASC
	`, sessionID)

	return ExecuteSelectQuery(query)
}

func (s *QueryExecutor) QueryLap(ctx context.Context, sessionID string, lapID string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		SELECT * FROM TelemetryTicks
		WHERE session_name = 'RACE' AND session_id = '%s' AND lap_id = '%s'
		ORDER BY timestamp DESC
	`, sessionID, lapID)

	return ExecuteSelectQuery(query)
}
