package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strconv"

	"github.com/ojparkinson/telemetryService/internal/geojson"
	"github.com/ojparkinson/telemetryService/internal/messaging"
	"github.com/ojparkinson/telemetryService/internal/persistance"
	"github.com/ojparkinson/telemetryService/internal/sync"
	qdb "github.com/questdb/go-questdb-client/v4"
	"google.golang.org/protobuf/proto"
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

	options := geojson.ConversionOptions{}

	lapData, err := s.queryExecutor.QueryLap(r.Context(), sessionID, lapID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch lap data")
		return
	}

	geoJSON, err := geojson.ConvertToGeoJSON(lapData, options)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to convert to GeoJSON")
		return
	}

	respondGzipJSON(w, http.StatusOK, geoJSON)
}

// /api/sync/lap/{sessionId}/{lapId}
func (s *Server) handleSyncLap(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")
	lapID := r.PathValue("lapId")

	sessionData, err := s.queryExecutor.QueryGeneralLap(r.Context(), sessionID, lapID)
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

	w.WriteHeader(200)
}

// /api/ingest
func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.Header.Get("content-type") == "application/x-protobuf" {
		ctx := context.TODO()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		batch := &messaging.TelemetryBatch{}
		err = proto.Unmarshal(body, batch)
		if err != nil {
			w.WriteHeader(400)
			return
		}

		sender, err := qdb.NewLineSender(
			context.Background(),
			qdb.WithHttp(),
			qdb.WithAddress(fmt.Sprintf("%s:9000", s.config.QuestDbHost)),
			qdb.WithInitBufferSize(2*1024*1024), // 2MB initial buffer (default: 128KB)
		)
		defer sender.Close(ctx)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		persistance.WriteBatch(sender, batch.Records)

		w.WriteHeader(200)
	} else {

		w.WriteHeader(400)
	}
}
