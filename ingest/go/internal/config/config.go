package config

import (
	"log"
	"os"
	"runtime"
	"strconv"
	"time"
)

type Config struct {
	WorkerCount   int
	FileQueueSize int
	WorkerTimeout time.Duration

	BatchSizeBytes int
	BatchTimeout   time.Duration
	MaxRetries     int
	RetryDelay     time.Duration

	FileAgeThreshold   time.Duration
	FileProcessTimeout time.Duration

	GoMaxProcs int

	ServerUrl string

	IngestURL string

	Compression string

	ShutdownTimeout time.Duration

	BatchSizeRecords int

	DryRun bool

	DataDirectory string
}

func LoadConfig() *Config {
	cpuCount := runtime.GOMAXPROCS(0)

	workerCount := getEnvAsInt("WORKER_COUNT", cpuCount)

	serverUrl := getEnv("SERVER_URL", "http://localhost")

	return &Config{
		WorkerCount:   workerCount,
		FileQueueSize: getEnvAsInt("FILE_QUEUE_SIZE", 1000),
		WorkerTimeout: getEnvAsDuration("WORKER_TIMEOUT", 30*time.Minute),

		BatchSizeBytes: getEnvAsInt("BATCH_SIZE_BYTES", 33554432),
		BatchTimeout:   getEnvAsDuration("BATCH_TIMEOUT", 50*time.Millisecond),
		MaxRetries:     getEnvAsInt("MAX_RETRIES", 3),
		RetryDelay:     getEnvAsDuration("RETRY_DELAY", 250*time.Millisecond),

		FileAgeThreshold:   getEnvAsDuration("FILE_AGE_THRESHOLD", 30*time.Second),
		FileProcessTimeout: getEnvAsDuration("FILE_PROCESS_TIMEOUT", 10*time.Minute),

		GoMaxProcs: getEnvAsInt("GOMAXPROCS", cpuCount),

		ServerUrl: serverUrl,
		IngestURL: getEnv("INGEST_URL", serverUrl+":8010/api/ingest"),

		Compression:     wireCompression(),
		ShutdownTimeout: getEnvAsDuration("SHUTDOWN_TIMEOUT", 30*time.Second),

		DryRun: getEnvAsBool("DRY_RUN", false),

		BatchSizeRecords: getEnvAsInt("BATCH_SIZE_RECORDS", 24000),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return fallback
}

func wireCompression() string {
	switch v := getEnv("WIRE_COMPRESSION", "zstd"); v {
	case "zstd", "none":
		return v
	default:
		log.Printf("config: unrecognised WIRE_COMPRESSION %q, sending uncompressed", v)
		return "none"
	}
}
