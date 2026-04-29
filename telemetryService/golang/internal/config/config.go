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
}

func NewConfig() *Config {
	questdbHost := getEnv("QUESTDB_HOST", "localhost")
	questdbPort := getEnvInt("QUESTDB_PORT", 9002)

	return &Config{
		QuestDbHost:   questdbHost,
		QuestDBPort:   questdbPort,
		QuestPoolSize: getSenderPool(),
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

	cpuCount := runtime.GOMAXPROCS(0)
	return max(cpuCount, 2)
}
