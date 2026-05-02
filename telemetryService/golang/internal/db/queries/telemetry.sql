-- name: GetLapTelemetry :many
SELECT session_num, track_name, session_time,
         speed, throttle, brake, rpm, gear, lap_dist_pct,
         steering_wheel_angle, lat, lon,
         velocity_x, velocity_y, velocity_z,
         lat_accel,
         fuel_level, lap_current_lap_time, player_car_position
FROM TelemetryTicks
WHERE session_name = 'RACE' AND session_id = $1 AND lap_id = $2
ORDER BY session_time ASC;
