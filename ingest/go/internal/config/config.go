package config

import (
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

	RabbitMQURL     string
	DisableRabbitMQ bool

	FileAgeThreshold   time.Duration
	FileProcessTimeout time.Duration

	GoMaxProcs int

	EnablePprof  bool
	PprofPort    string
	MemoryTuning bool

	IngestUrl string

	BatchSizeRecords int

	UseStructPipeline bool

	// Data directory configuration
	DataDirectory string

	CFAccountID    string
	CFD1DatabaseID string
	CFApiToken     string
	R2AccountID    string
	R2AccessKeyID  string
	R2SecretAccess string
	R2BucketNme    string
}

func LoadConfig() *Config {
	cpuCount := runtime.NumCPU()

	defaultWorkerCount := cpuCount + (cpuCount / 4)
	if defaultWorkerCount < 4 {
		defaultWorkerCount = 4
	}

	defaultGoMaxProcs := 0

	workerCount := getEnvAsInt("WORKER_COUNT", defaultWorkerCount)

	return &Config{
		WorkerCount:   workerCount,
		FileQueueSize: getEnvAsInt("FILE_QUEUE_SIZE", 1000),
		WorkerTimeout: getEnvAsDuration("WORKER_TIMEOUT", 30*time.Minute),

		BatchSizeBytes: getEnvAsInt("BATCH_SIZE_BYTES", 33554432),
		BatchTimeout:   getEnvAsDuration("BATCH_TIMEOUT", 50*time.Millisecond),
		MaxRetries:     getEnvAsInt("MAX_RETRIES", 3),
		RetryDelay:     getEnvAsDuration("RETRY_DELAY", 250*time.Millisecond),

		DisableRabbitMQ: getEnvAsBool("DISABLE_RABBITMQ", false),
		RabbitMQURL:     getEnv("RABBITMQ_URL", "amqp://admin:changeme@localhost:5672"),

		FileAgeThreshold:   getEnvAsDuration("FILE_AGE_THRESHOLD", 30*time.Second),
		FileProcessTimeout: getEnvAsDuration("FILE_PROCESS_TIMEOUT", 10*time.Minute),

		GoMaxProcs: getEnvAsInt("GOMAXPROCS", defaultGoMaxProcs),

		// Development & Monitoring
		EnablePprof:  getEnvAsBool("ENABLE_PPROF", false),
		PprofPort:    getEnv("PPROF_PORT", "6060"),
		MemoryTuning: getEnvAsBool("MEMORY_TUNING", true),

		IngestUrl: getEnv("INGEST_URL", "http://localhost:8010/api/ingest"),

		UseStructPipeline: getEnvAsBool("USE_STRUCT_PIPELINE", true),

		// Record Processing
		BatchSizeRecords: getEnvAsInt("BATCH_SIZE_RECORDS", 16000),

		CFAccountID:    getEnv("CF_ACCOUNT_ID", ""),
		CFD1DatabaseID: getEnv("CF_D1_DATABASE_ID", ""),
		CFApiToken:     getEnv("CF_API_TOKEN", ""),
		R2AccountID:    getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:  getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccess: getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketNme:    getEnv("R2_BUCKET_NAME", ""),

		// Data Directory - defaults to ./ibt_files/ for backward compatibility
		// DataDirectory: getEnv("IBT_DATA_DIR", "./ibt_files/"),
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
