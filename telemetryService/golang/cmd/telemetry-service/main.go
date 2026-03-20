package main

import (
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/ojparkinson/telemetryService/internal/api"
	"github.com/ojparkinson/telemetryService/internal/config"
	"github.com/ojparkinson/telemetryService/internal/metrics"
	"github.com/ojparkinson/telemetryService/internal/persistance"
	"github.com/ojparkinson/telemetryService/internal/queue"
)

func main() {
	log.Println("Starting telemetry service")

	config := config.NewConfig()

	// Create database schema
	schema := persistance.NewSchema(config)
	if err := schema.CreateTableHTTP(); err != nil {
		log.Printf("Failed to create table: %v", err)
		log.Println("Exiting due to database initialization failure")
		os.Exit(1)
	}
	log.Println("Database schema initialized successfully")

	apiServer := api.NewServer(":8010", &persistance.QueryExecutor{})

	log.Println("creating server")
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	// Create sender pool
	senderPool, err := persistance.NewSenderPool(config)
	if err != nil {
		log.Printf("Failed to create sender pool: %v", err)
		log.Println("Exiting due to sender pool initialization failure")
		os.Exit(1)
	}
	log.Println("Sender pool created successfully")

	// Start Prometheus metrics server
	go metrics.MetricsHandler()
	log.Println("Starting to consume messages from RabbitMQ")

	// Start message queue subscriber
	messaging := queue.NewSubscriber(senderPool)
	go func() {
		messaging.Subscribe(config)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down...")
}
