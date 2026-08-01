package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ojparkinson/telemetryService/internal/domain"
	"github.com/ojparkinson/telemetryService/internal/geojson"
	"github.com/ojparkinson/telemetryService/internal/messaging"
	"github.com/ojparkinson/telemetryService/internal/sync"
	"google.golang.org/protobuf/proto"
)

// /api/sessions
func (s *Server) handleGetSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.sessions.ListSessions(r.Context())
	if err != nil {
		log.Println(err)
		respondError(w, http.StatusInternalServerError, "Failed to fetch sessions")
		return
	}

	respondJSON(w, 200, sessions)
}

// /api/sessions/123456/laps
func (s *Server) handleGetLaps(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		respondError(w, http.StatusInternalServerError, "Invalid session ID")
		return
	}

	rows, err := s.sessions.ListLaps(r.Context(), sessionID)
	if err != nil {
		log.Println(err)
		respondError(w, http.StatusInternalServerError, "Failed to fetch laps")
		return
	}

	lapIDs := make([]int, len(rows))

	for i, row := range rows {
		lapIDs[i] = row.LapID
	}

	respondJSON(w, 200, lapIDs)

}

type telemetryPointWire struct {
	SessionTime        float64 `json:"session_time"`
	Speed              float64 `json:"speed"`
	RPM                float64 `json:"rpm"`
	Throttle           float64 `json:"throttle"`
	Brake              float64 `json:"brake"`
	Gear               uint32  `json:"gear"`
	LapDistPct         float64 `json:"lap_dist_pct"`
	SteeringWheelAngle float64 `json:"steering_wheel_angle"`
	PlayerCarPosition  float64 `json:"player_car_position"`
	Lat                float64 `json:"lat"`
	Lon                float64 `json:"lon"`
	VelocityX          float64 `json:"velocity_x"`
	VelocityY          float64 `json:"velocity_y"`
	VelocityZ          float64 `json:"velocity_z"`
	LatAccel           float64 `json:"lat_accel"`
	LapCurrentLapTime  float64 `json:"lap_current_lap_time"`
	FuelLevel          float64 `json:"fuel_level"`
}

type telemetryEnvelope struct {
	TrackName  string               `json:"track_name"`
	SessionNum string               `json:"session_num"`
	Points     []telemetryPointWire `json:"points"`
}

// /api/sessions/123456/laps/1
func (s *Server) handleGetTelemetry(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	lapID := chi.URLParam(r, "lapId")
	if sessionID == "" || lapID == "" {
		respondError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	lapData, err := s.sessions.GetLapTelemetry(r.Context(), sessionID, lapID)
	if err != nil {
		log.Println(err)
		respondError(w, http.StatusInternalServerError, "Failed to fetch lap data")
		return
	}

	envelope := telemetryEnvelope{Points: make([]telemetryPointWire, len(lapData))}
	if len(lapData) > 0 {
		envelope.TrackName = lapData[0].TrackName
		envelope.SessionNum = lapData[0].SessionNum
	}
	for i, p := range lapData {
		envelope.Points[i] = telemetryPointWire{
			SessionTime:        p.SessionTime,
			Speed:              p.Speed,
			RPM:                p.RPM,
			Throttle:           p.Throttle,
			Brake:              p.Brake,
			Gear:               p.Gear,
			LapDistPct:         p.LapDistPct,
			SteeringWheelAngle: p.SteeringWheelAngle,
			PlayerCarPosition:  p.PlayerCarPosition,
			Lat:                p.Lat,
			Lon:                p.Lon,
			VelocityX:          p.VelocityX,
			VelocityY:          p.VelocityY,
			VelocityZ:          p.VelocityZ,
			LatAccel:           p.LatAccel,
			LapCurrentLapTime:  p.LapCurrentLapTime,
			FuelLevel:          p.FuelLevel,
		}
	}

	respondJSON(w, 200, envelope)
}

// /api/sessions/123456/laps/1/geojson
func (s *Server) handleGetTelemetryGeoJson(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	lapID := chi.URLParam(r, "lapId")

	options := geojson.ConversionOptions{}

	lapData, err := s.sessions.GetLapTelemetry(r.Context(), sessionID, lapID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch lap data")
		return
	}

	geoJSON, err := geojson.ConvertToGeoJSON(lapData, options)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to convert to GeoJSON")
		return
	}

	respondJSON(w, http.StatusOK, geoJSON)

}

// /api/sync/lap/{sessionId}/{lapId}
func (s *Server) handleSyncLap(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	lapID := chi.URLParam(r, "lapId")

	sessionData, err := s.sessions.GetLapTelemetry(r.Context(), sessionID, lapID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch lap data")
		return
	}

	data, _ := json.Marshal(sessionData)

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write(data)
	gz.Close()

	sync.SyncLap(sessionData)

	w.WriteHeader(http.StatusOK)

}

// /api/ingest
func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") == "application/x-protobuf" {

		body, err := io.ReadAll(r.Body)
		if err != nil {
			respondError(w, 500, fmt.Sprintf("failed to read body: %w", err))
			return
		}
		defer r.Body.Close()

		batch := &messaging.TelemetryBatch{}
		if err := proto.Unmarshal(body, batch); err != nil {
			respondError(w, http.StatusBadRequest, "Failed to fetch lap data")
			return
		}

		points := make([]*domain.TelemetryPoint, len(batch.Records))
		for i, record := range batch.Records {
			singlePoint := domain.TelemetryPointFromProto(record, batch.SessionId, batch.CarId)
			points[i] = &singlePoint
		}

		err = s.writer.WriteBatch(r.Context(), points)
		if err != nil {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
