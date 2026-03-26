-- name: ListSessions :many
SELECT DISTINCT session_id, track_name, session_name, MAX(CAST(lap_id as int)) as max_lap_id, MAX(timestamp) as last_updated
FROM TelemetryTicks
WHERE session_name = 'RACE' AND lap_id > 0
GROUP BY session_id, track_name, session_name
ORDER BY last_updated DESC;