package api

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/andybalholm/brotli"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ojparkinson/telemetryService/internal/domain"
)

type Server struct {
	sessions domain.SessionRepository
	writer   domain.TelemetryWriter
	logger   *log.Logger
	addr     string
	app      *chi.Mux
}

func NewServer(addr string, sessions domain.SessionRepository, writer domain.TelemetryWriter) *Server {
	app := chi.NewRouter()

	compressor := middleware.NewCompressor(5)
	compressor.SetEncoder("br", func(w io.Writer, level int) io.Writer {
		return brotli.NewWriterLevel(w, level)
	})
	app.Use(compressor.Handler)
	app.Use(middleware.Logger)

	server := &Server{
		sessions: sessions,
		writer:   writer,
		logger:   log.New(os.Stdout, "[API] ", log.LstdFlags),
		addr:     addr,
		app:      app,
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
