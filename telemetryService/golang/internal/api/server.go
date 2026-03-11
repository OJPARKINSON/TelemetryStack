package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/ojparkinson/telemetryService/internal/config"
	"github.com/ojparkinson/telemetryService/internal/persistance"
)

type Server struct {
	httpServer    *http.Server
	logger        *log.Logger
	queryExecutor *persistance.QueryExecutor
	config        *config.Config
	senderPool    *persistance.SenderPool

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
	}

	server.httpServer = &http.Server{
		Addr:    addr,
		Handler: server.setupRoutes(),
	}

	return server
}

func (s *Server) Start() error {
	s.logger.Printf("Starting api server on: %s", s.httpServer.Addr)
	if err := s.app.Listen(s.httpServer.Addr); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Println("Shutting down admin server...")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) setupRoutes() http.Handler {
	mux := http.NewServeMux()

	s.app.Post("/api/ingest", s.handleIngest)

	s.app.Get("/api/sessions", s.handleGetSessions)
	s.app.Get("/api/sessions/:sessionId/laps", s.handleGetLaps)
	s.app.Get("/api/sessions/:sessionId/laps/:lapId", s.handleGetTelemetry)
	s.app.Get("/api/sessions/:sessionId/laps/:lapId/geojson", s.handleGetTelemetryGeoJson)

	// Add panic recovery middleware
	return RecoveryMiddleware(mux)
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				fmt.Println(err)
				debug.PrintStack() // from "runtime/debug"
				// app.serverError(w, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
