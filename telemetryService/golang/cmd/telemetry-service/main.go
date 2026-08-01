package main

import (
	"context"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ojparkinson/telemetryService/internal/adapter/questdb"
	"github.com/ojparkinson/telemetryService/internal/api"
	"github.com/ojparkinson/telemetryService/internal/config"
	"github.com/ojparkinson/telemetryService/internal/metrics"
	"github.com/ojparkinson/telemetryService/internal/queue"
)

func main() {
	log.Println("Starting telemetry service")

	cfg := config.NewConfig()

	schema := questdb.NewSchema(cfg.QuestDbHost, cfg.QuestDBPort)
	if err := schema.CreateTableHTTP(); err != nil {
		log.Printf("Failed to create table: %v", err)
		log.Println("Exiting due to database initialization failure")
		os.Exit(1)
	}
	log.Println("Database schema initialized successfully")

	senderPool, err := questdb.NewSenderPool(cfg)
	if err != nil {
		log.Printf("Failed to create sender pool: %v", err)
		log.Println("Exiting due to sender pool initialization failure")
		os.Exit(1)
	}
	log.Println("Sender pool created successfully")

	config, err := pgxpool.ParseConfig(fmt.Sprintf("postgres://admin:quest@%s:%d/questdb?sslmode=disable", cfg.QuestDbHost, 8812))
	if err != nil {
		log.Fatalf("pgx config fail: %v", err)
	}
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec

	pgxPool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("pgx pool: %v", err)
	}
	defer pgxPool.Close()

	repo := questdb.NewRepository(pgxPool, senderPool)

	ingestQueue := queue.NewQueue(repo)

	ingestQueue.StartRouter()
	for i := 0; i < ingestQueue.Workers; i++ {
		ingestQueue.StartWorker(i)
	}
	log.Println("Telemetry queue created successfully")

	apiServer := api.NewServer(":8010", repo, ingestQueue)

	log.Println("creating server")
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	// Start Prometheus metrics server
	go metrics.MetricsHandler()
	log.Println("Starting to consume tick batches")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	ingestQueue.Shutdown()
	log.Println("Shutting down...")
}
