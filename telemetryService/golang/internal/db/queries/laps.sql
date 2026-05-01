-- name: ListLaps :many
SELECT DISTINCT CAST(lap_id as int) FROM TelemetryTicks
WHERE session_id = $1
ORDER BY CAST(lap_id as INT);