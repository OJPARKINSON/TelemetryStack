package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/ojparkinson/telemetryService/internal/geojson"
	"github.com/ojparkinson/telemetryService/internal/messaging"
	"github.com/ojparkinson/telemetryService/internal/persistance"
	"github.com/ojparkinson/telemetryService/internal/sync"
	"google.golang.org/protobuf/proto"
)

// /api/sessions
func (s *Server) handleGetSessions(c fiber.Ctx) {
	sessions, err := s.queryExecutor.QuerySessions()
	if err != nil {
		log.Println(err)
		respondError(c, "Failed to fetch sessions", http.StatusInternalServerError)
		return
	}

	respondJSON(c, 200, sessions)
}

// /api/sessions/123456/laps
func (s *Server) handleGetLaps(c fiber.Ctx) {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		respondError(c, "Invalid session ID", http.StatusBadRequest)
		return
	}

	rows, err := s.queryExecutor.QueryLaps(c.Context(), sessionID)
	if err != nil {
		log.Println(err)
		respondError(c, "Failed to fetch laps", http.StatusInternalServerError)
		return
	}

	laps := make([]int, len(rows))
	for i, row := range rows {
		laps[i], _ = strconv.Atoi(row["lap_id"].(string))
	}

	slices.Sort(laps)

	respondJSON(c, 200, laps)
}

// /api/sessions/123456/laps/1
func (s *Server) handleGetTelemetry(c fiber.Ctx) {
	sessionID := c.Params("sessionId")
	lapID := c.Params("lapId")
	if sessionID == "" || lapID == "" {
		respondError(c, "Invalid session ID", http.StatusBadRequest)
		return
	}

	lapData, err := s.queryExecutor.QueryLap(c.Context(), sessionID, lapID)
	if err != nil {
		log.Println(err)
		respondError(c, "Failed to fetch lap data", http.StatusInternalServerError)
		return
	}

	respondJSON(c, 200, lapData)
}

// /api/sessions/123456/laps/1/geojson
func (s *Server) handleGetTelemetryGeoJson(c fiber.Ctx) {
	sessionID := c.Params("sessionId")
	lapID := c.Params("lapId")

	options := geojson.ConversionOptions{}

	lapData, err := s.queryExecutor.QueryLap(c.Context(), sessionID, lapID)
	if err != nil {
		respondError(c, "Failed to fetch lap data", http.StatusInternalServerError)
		return
	}

	geoJSON, err := geojson.ConvertToGeoJSON(lapData, options)
	if err != nil {
		respondError(c, "Failed to convert to GeoJSON", http.StatusInternalServerError)
		return
	}

	respondJSON(c, http.StatusOK, geoJSON)
}

// /api/sync/lap/{sessionId}/{lapId}
func (s *Server) handleSyncLap(c fiber.Ctx) {
	sessionID := c.Params("sessionId")
	lapID := c.Params("lapId")

	sessionData, err := s.queryExecutor.QueryGeneralLap(c.Context(), sessionID, lapID)
	if err != nil {
		respondError(c, "Failed to fetch lap data", http.StatusInternalServerError)
		return
	}

	data, _ := json.Marshal(sessionData)

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write(data)
	gz.Close()

	sync.SyncLap(sessionData)

	c.Status(200)
}

// /api/ingest
func (s *Server) handleIngest(c fiber.Ctx) {
	if c.Get("content-type") == "application/x-protobuf" {
		batch := &messaging.TelemetryBatch{}
		err := proto.Unmarshal(c.Body(), batch)
		if err != nil {
			respondError(c, "Failed to fetch lap data", http.StatusBadRequest)
			return
		}
		sender := s.senderPool.Get()
		defer s.senderPool.Return(sender)

		persistance.WriteBatch(sender, batch.Records)

		c.Status(http.StatusOK)
	} else {
		c.Status(http.StatusInternalServerError)
	}
}
