package questdb

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Schema struct {
	host string
	port int
}

func NewSchema(host string, port int) *Schema {
	return &Schema{host: host, port: port}
}

// todo: add retry logic to stop it falling over on the pi
func (s *Schema) CreateTableHTTP() error {
	sql := `
		    CREATE TABLE IF NOT EXISTS TelemetryTicks (
                session_id SYMBOL CAPACITY 50000 INDEX,
                track_name SYMBOL CAPACITY 100 INDEX,
                track_id SYMBOL CAPACITY 100 INDEX,
                lap_id SYMBOL CAPACITY 500,
                session_num SYMBOL CAPACITY 20,
                session_type SYMBOL CAPACITY 10 INDEX,
                session_name SYMBOL CAPACITY 50 INDEX,
                car_id SYMBOL CAPACITY 1000 INDEX,
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
            WITH maxUncommittedRows=1000000
            DEDUP UPSERT KEYS(timestamp, session_id);
	`
	err := s.execDDL(sql)
	return err
}

func (s *Schema) AddIndexes() error {
	indexes := []string{
		"ALTER TABLE TelemetryTicks ADD INDEX session_lap_idx (session_id, lap_id);",
		"ALTER TABLE TelemetryTicks ADD INDEX track_session_idx (track_name, session_id);",
		"ALTER TABLE TelemetryTicks ADD INDEX session_time_idx (session_id, session_time);",
	}

	for _, idx := range indexes {
		if err := s.execDDL(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
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
