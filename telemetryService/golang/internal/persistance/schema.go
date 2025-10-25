package persistance

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ojparkinson/telemetryService/internal/config"
)

type Schema struct {
	config *config.Config
}

func NewSchema(config *config.Config) *Schema {
	return &Schema{
		config: config,
	}
}

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
                throttle FLOAT,
                brake FLOAT,
                steering_wheel_angle FLOAT,
                rpm FLOAT,
                velocity_x FLOAT,
                velocity_y FLOAT,
                velocity_z FLOAT,
                fuel_level FLOAT,
                alt FLOAT,
                lat_accel FLOAT,
                long_accel FLOAT,
                vert_accel FLOAT,
                pitch FLOAT,
                roll FLOAT,
                yaw FLOAT,
                yaw_north FLOAT,
                voltage FLOAT,
                waterTemp FLOAT,
                lFpressure FLOAT,
                rFpressure FLOAT,
                lRpressure FLOAT,
                rRpressure FLOAT,
                lFtempM FLOAT,
                rFtempM FLOAT,
                lRtempM FLOAT,
                rRtempM FLOAT,
                timestamp TIMESTAMP
            ) TIMESTAMP(timestamp) PARTITION BY DAY 
            WAL
            WITH maxUncommittedRows=1000000
            DEDUP UPSERT KEYS(timestamp, session_id);
	`

	return s.executeQuery(sql)
}

func (s *Schema) AddIndexes() error {
	indexes := []string{
		"ALTER TABLE TelemetryTicks ADD INDEX session_lap_idx (session_id, lap_id);",
		"ALTER TABLE TelemetryTicks ADD INDEX track_session_idx (track_name, session_id);",
		"ALTER TABLE TelemetryTicks ADD INDEX session_time_idx (session_id, session_time);",
	}

	for _, idx := range indexes {
		if err := s.executeQuery(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

func (s *Schema) executeQuery(query string) error {
	maxRetries := 10
	baseDelay := 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := http.Get(
			fmt.Sprintf("http://%s:%d/exec?query=%s",
				s.config.QuestDbHost,
				s.config.QuestDBPort,
				url.QueryEscape(query)),
		)

		if err != nil {
			if attempt < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<uint(attempt))
				fmt.Printf("QuestDB connection failed (attempt %d/%d), retrying in %v: %v\n", attempt+1, maxRetries, delay, err)
				time.Sleep(delay)
				continue
			}
			fmt.Println("failed to execute query after all retries: ", err)
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusServiceUnavailable {
			if attempt < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<uint(attempt))
				fmt.Printf("QuestDB not ready (status %d, attempt %d/%d), retrying in %v\n", resp.StatusCode, attempt+1, maxRetries, delay)
				time.Sleep(delay)
				continue
			}
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("query failed with status %d", resp.StatusCode)
		}

		fmt.Println("Created table")
		return nil
	}

	return fmt.Errorf("failed to execute query after %d retries", maxRetries)
}
