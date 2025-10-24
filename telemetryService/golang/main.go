package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ojparkinson/telemetryService/internal/config"
	"github.com/ojparkinson/telemetryService/internal/persistance"
	"github.com/ojparkinson/telemetryService/internal/queue"
)

func main() {
	fmt.Println("Starting subscriber")

	questdbHost := getEnv("QUESTDB_HOST", "localhost")
	questdbPort := getEnvInt("QUESTDB_PORT", 9000)
	poolSize := getEnvInt("SENDER_POOL_SIZE", 10)

	config := config.NewConfig()

	senderPool, err := persistance.NewSenderPool(poolSize, questdbHost, questdbPort)
	if err != nil {
		fmt.Println("Failed to create sender pool: ", err)
	}
	messaging := queue.NewSubscriber(senderPool)

	fmt.Println("Starting to consume")
	messaging.Subscribe(config)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
