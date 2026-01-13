package api

import (
	"log"
	"net/http"
	"slices"
	"strconv"

	"github.com/ojparkinson/telemetryService/internal/geojson"
)

// /api/sessions
func (s *Server) handleGetSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.queryExecutor.QuerySessions(r.Context())
	if err != nil {
		log.Println(err)
		respondError(w, http.StatusInternalServerError, "Failed to fetch sessions")
		return
	}

	respondJSON(w, 200, sessions)
}

// /api/sessions/123456/laps
func (s *Server) handleGetLaps(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")
	if sessionID == "" {
		respondError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	rows, err := s.queryExecutor.QueryLaps(r.Context(), sessionID)
	if err != nil {
		log.Println(err)
		respondError(w, http.StatusInternalServerError, "Failed to fetch laps")
		return
	}

	laps := make([]int, len(rows))
	for i, row := range rows {
		laps[i], _ = strconv.Atoi(row["lap_id"].(string))
	}

	slices.Sort(laps)

	respondJSON(w, 200, laps)
}

// /api/sessions/123456/laps/1
func (s *Server) handleGetTelemetry(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")
	lapID := r.PathValue("lapId")
	if sessionID == "" || lapID == "" {
		respondError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	lapData, err := s.queryExecutor.QueryLap(r.Context(), sessionID, lapID)
	if err != nil {
		log.Println(err)
		respondError(w, http.StatusInternalServerError, "Failed to fetch lap data")
		return
	}

	respondGzipJSON(w, 200, lapData)
}

// /api/sessions/123456/laps/1/geojson
func (s *Server) handleGetTelemetryGeoJson(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")
	lapID := r.PathValue("lapId")
	if sessionID == "" || lapID == "" {
		respondError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	lapData, err := s.queryExecutor.QueryLap(r.Context(), sessionID, lapID)
	if err != nil {
		log.Println(err)
		respondError(w, http.StatusInternalServerError, "Failed to fetch lap data")
		return
	}

	col, _ := geojson.ConvertToGeoJSON(lapData, geojson.ConversionOptions{})

	respondGzipJSON(w, 200, col)
}
