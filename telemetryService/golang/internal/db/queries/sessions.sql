-- name: ListSessions :many
SELECT session_id, track_name, session_name, MAX(lap_id) as max_lap_id, MAX(timestamp) as last_updated FROM TelemetryTicks
WHERE session_name = 'RACE' AND lap_id > 0
GROUP BY session_id, track_name, session_name
ORDER BY last_updated DESC;