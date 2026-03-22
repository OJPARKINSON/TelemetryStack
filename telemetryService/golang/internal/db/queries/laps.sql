-- name: ListLaps :many
SELECT DISTINCT lap_id FROM TelemetryTicks
WHERE session_id = $1 ORDER BY CAST(lap_id as int) ASC;
