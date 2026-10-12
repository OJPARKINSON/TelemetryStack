-- name: UpsertTrack :exec
INSERT INTO Tracks (track_id, track_name, sector_start_pct, ts)
VALUES ($1, $2, $3, '1970-01-01T00:00:00.000000Z');

-- name: GetTrack :one
SELECT track_id, track_name, sector_start_pct FROM Tracks
WHERE track_id = $1;

-- name: GetLapTrackID :one
SELECT track_id FROM TelemetryTicks
WHERE session_id = $1 AND lap_id = $2
LIMIT 1;
