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

	respondJSON(w, 200, rows)

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

	respondJSON(w, 200, lapData)

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

		s.writer.WriteBatch(r.Context(), batch.Records)

		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
