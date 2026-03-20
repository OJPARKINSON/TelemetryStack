package api

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/andybalholm/brotli"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ojparkinson/telemetryService/internal/config"
	"github.com/ojparkinson/telemetryService/internal/persistance"
)

type Server struct {
	config        *config.Config
	logger        *log.Logger
	queryExecutor *persistance.QueryExecutor
	senderPool    *persistance.SenderPool
	addr          string

	app *chi.Mux
}

func NewServer(addr string, config *config.Config, senderPool *persistance.SenderPool) *Server {
	r := chi.NewRouter()

	compressor := middleware.NewCompressor(5)
	compressor.SetEncoder("br", func(w io.Writer, level int) io.Writer {
		return brotli.NewWriterLevel(w, level)
	})
	r.Use(middleware.Logger)

	server := &Server{
		queryExecutor: &persistance.QueryExecutor{Config: config},
		logger:        log.New(os.Stdout, "[API] ", log.LstdFlags),
		config:        config,
		senderPool:    senderPool,
		app:           r,
		addr:          addr,
	}

	server.setupRoutes()

	return server
}

func (s *Server) Start() error {
	s.logger.Printf("Starting api server on: %s", s.addr)
	if err := http.ListenAndServe(s.addr, s.app); err != nil {
		log.Fatal(err)
	}
	return nil
}

// func (s *Server) Shutdown(ctx context.Context) error {
// 	s.logger.Println("Shutting down admin server...")
// 	return s.app()
// }

func (s *Server) setupRoutes() {
	s.app.Post("/api/ingest", s.handleIngest)

	s.app.Get("/api/sessions", s.handleGetSessions)
	s.app.Get("/api/sessions/{sessionId}/laps", s.handleGetLaps)
	s.app.Get("/api/sessions/{sessionId}/laps/{lapId}", s.handleGetTelemetry)
	s.app.Get("/api/sessions/{sessionId}/laps/{lapId}/geojson", s.handleGetTelemetryGeoJson)
}
