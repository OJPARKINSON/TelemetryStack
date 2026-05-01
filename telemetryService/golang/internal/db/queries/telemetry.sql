-- name: GetLapTelemetry :many
SELECT session_id, lap_id, session_num, track_name, session_time,
         speed, throttle, brake, rpm, gear, lap_dist_pct,
         steering_wheel_angle, lat, lon, alt,
         velocity_x, velocity_y, velocity_z,
         lat_accel, long_accel, vert_accel,
         pitch, roll, yaw, yaw_north,
         fuel_level, lap_current_lap_time, player_car_position,
timestamp FROM TelemetryTicks
WHERE session_name = 'RACE' AND session_id = $1 AND lap_id = $2
ORDER BY timestamp ASC;