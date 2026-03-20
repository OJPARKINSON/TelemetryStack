package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/ojparkinson/telemetryService/internal/persistance"
)

type Server struct {
	httpServer    *http.Server
	logger        *log.Logger
	queryExecutor *persistance.QueryExecutor
}

func NewServer(addr string, queryExecutor *persistance.QueryExecutor) *Server {
	server := &Server{
		queryExecutor: queryExecutor,
		logger:        log.New(os.Stdout, "[API] ", log.LstdFlags),
	}

	server.httpServer = &http.Server{
		Addr:    addr,
		Handler: server.setupRoutes(),
	}

	return server
}

func (s *Server) Start() error {
	s.logger.Printf("Starting api server on: %s", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

	mux.HandleFunc("GET /api/sessions", s.handleGetSessions)
	mux.HandleFunc("GET /api/sessions/{sessionId}/laps", s.handleGetLaps)
	mux.HandleFunc("GET /api/sessions/{sessionId}/laps/{lapId}", s.handleGetTelemetry)
	mux.HandleFunc("GET /api/sessions/{sessionId}/laps/{lapId}/geojson", s.handleGetTelemetryGeoJson)

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
