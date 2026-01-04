package main

import (
	"log"
	"os"

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
	messaging.Subscribe(config)
}
