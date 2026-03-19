-- name: GetLapTelemetry :many
SELECT * FROM TelemetryTicks
WHERE session_name = 'RACE' AND session_id = $1 AND lap_id = $2
ORDER BY timestamp ASC;