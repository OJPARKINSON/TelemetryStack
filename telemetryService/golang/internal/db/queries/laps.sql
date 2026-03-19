-- name: ListLaps :many
SELECT DISTINCT lap_id FROM TelemetryTicks
WHERE session_id = @session_id ORDER BY lap_id ASC;