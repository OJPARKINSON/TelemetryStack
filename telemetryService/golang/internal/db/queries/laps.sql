-- name: ListLaps :many
SELECT DISTINCT CAST(lap_id AS INT) AS lap_id FROM TelemetryTicks
WHERE session_id = $1
ORDER BY lap_id;