package config

import (
	"os"
	"runtime"
	"strconv"
)

type Config struct {
	QuestDbHost   string
	QuestDBPort   int
	QuestPoolSize int
	RabbitMQHost  string
}

func NewConfig() *Config {
	questdbHost := getEnv("QUESTDB_HOST", "localhost")
	questdbPort := getEnvInt("QUESTDB_PORT", 9000)
	rabbitMqHost := getEnv("RABBITMQ_HOST", "localhost")

	return &Config{
		QuestDbHost:   questdbHost,
		QuestDBPort:   questdbPort,
		QuestPoolSize: getSenderPool(),
		RabbitMQHost:  rabbitMqHost,
	}
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

func getSenderPool() int {
	if envValue := os.Getenv("SENDER_POOL_SIZE"); envValue != "" {
		intEnv, _ := strconv.Atoi(envValue)
		return intEnv
	}

	cpuCount := runtime.NumCPU()
	return min(cpuCount*2+cpuCount/2, 50)
}
