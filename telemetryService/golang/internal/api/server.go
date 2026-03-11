package api

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/ojparkinson/telemetryService/internal/config"
	"github.com/ojparkinson/telemetryService/internal/persistance"
)

type Server struct {
	config        *config.Config
	logger        *log.Logger
	queryExecutor *persistance.QueryExecutor
	senderPool    *persistance.SenderPool
	addr          string

	app *fiber.App
}

func NewServer(addr string, config *config.Config, senderPool *persistance.SenderPool) *Server {
	app := fiber.New()

	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed, // or LevelBestCompression, LevelDefault
	}))

	server := &Server{
		queryExecutor: &persistance.QueryExecutor{Config: config},
		logger:        log.New(os.Stdout, "[API] ", log.LstdFlags),
		config:        config,
		senderPool:    senderPool,
		app:           app,
		addr:          addr,
	}

	server.setupRoutes()

	return server
}

func (s *Server) Start() error {
	s.logger.Printf("Starting api server on: %s", s.addr)
	if err := s.app.Listen(s.addr); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Println("Shutting down admin server...")
	return s.app.Shutdown()
}

func (s *Server) setupRoutes() {
	s.app.Post("/api/ingest", s.handleIngest)

	s.app.Get("/api/sessions", s.handleGetSessions)
	s.app.Get("/api/sessions/:sessionId/laps", s.handleGetLaps)
	s.app.Get("/api/sessions/:sessionId/laps/:lapId", s.handleGetTelemetry)
	s.app.Get("/api/sessions/:sessionId/laps/:lapId/geojson", s.handleGetTelemetryGeoJson)
}
