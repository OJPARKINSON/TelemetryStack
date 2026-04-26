package questdb

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Schema struct {
	host string
	port int
}

func NewSchema(host string, port int) *Schema {
	return &Schema{host: host, port: port}
}

// CREATE TABLE IF NOT EXISTS TelemetryTicks (
//     session_id TEXT,
//     lap_id INT,
//     session_num TEXT,
//     session_name TEXT,
//     track_name TEXT,
//     session_type TEXT,
//     car_id TEXT,
//     track_id TEXT,
//     gear INT,

func (s *Schema) CreateTableHTTP() error {
	sql := `
		    CREATE TABLE IF NOT EXISTS TelemetryTicks (
                session_id SYMBOL CAPACITY 50000 INDEX,
                lap_id SYMBOL INDEX,
                session_num SYMBOL CAPACITY 20,
                session_name SYMBOL,
                track_name STRING,
                session_type STRING,
                car_id STRING,
                track_id STRING,
                gear INT,
                player_car_position INT,
                speed DOUBLE,
                lap_dist_pct DOUBLE,
                session_time DOUBLE,
                lat DOUBLE,
                lon DOUBLE,
                lap_current_lap_time DOUBLE,
                lapLastLapTime DOUBLE,
                lapDeltaToBestLap DOUBLE,
                throttle DOUBLE,
                brake DOUBLE,
                steering_wheel_angle DOUBLE,
                rpm DOUBLE,
                velocity_x DOUBLE,
                velocity_y DOUBLE,
                velocity_z DOUBLE,
                fuel_level DOUBLE,
                alt DOUBLE,
                lat_accel DOUBLE,
                long_accel DOUBLE,
                vert_accel DOUBLE,
                pitch DOUBLE,
                roll DOUBLE,
                yaw DOUBLE,
                yaw_north DOUBLE,
                voltage DOUBLE,
                waterTemp DOUBLE,
                lFpressure DOUBLE,
                rFpressure DOUBLE,
                lRpressure DOUBLE,
                rRpressure DOUBLE,
                lFtempM DOUBLE,
                rFtempM DOUBLE,
                lRtempM DOUBLE,
                rRtempM DOUBLE,
                timestamp TIMESTAMP
            ) TIMESTAMP(timestamp) PARTITION BY DAY
            WAL
            WITH maxUncommittedRows=50000
            DEDUP UPSERT KEYS(timestamp, session_id);
	`

	for i := 0; i < 3; i++ {
		err := s.execDDL(sql)
		if err == nil {
			return nil
		}
		if i == 2 {
			return err
		}
		time.Sleep(12 * time.Second)
	}

	return nil
}

func (s *Schema) execDDL(sql string) error {
	resp, err := http.Get(
		fmt.Sprintf("http://%s:%d/exec?query=%s", s.host, s.port,
			url.QueryEscape(sql)),
	)
	if err != nil {
		return fmt.Errorf("ddl exec: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ddl failed (%d): %s", resp.StatusCode, body)
	}
	return nil
}
