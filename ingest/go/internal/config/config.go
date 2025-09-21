package config

import (
	"os"
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

	RabbitMQPoolSize      int
	RabbitMQPrefetchCount int
	RabbitMQBatchSize     int
	RabbitMQBatchTimeout  time.Duration
	RabbitMQConfirms      bool
	RabbitMQPersistent    bool
	RabbitMQHeartbeat     time.Duration
	RabbitMQChannelMax    int
	RabbitMQFrameSize     int

	BatchSizeRecords int
}

func LoadConfig() *Config {
	return &Config{
		WorkerCount:   getEnvAsInt("WORKER_COUNT", 20),
		FileQueueSize: getEnvAsInt("FILE_QUEUE_SIZE", 1000),
		WorkerTimeout: getEnvAsDuration("WORKER_TIMEOUT", 30*time.Minute),

		BatchSizeBytes: getEnvAsInt("BATCH_SIZE_BYTES", 67108864),
		BatchTimeout:   getEnvAsDuration("BATCH_TIMEOUT", 10*time.Millisecond),
		MaxRetries:     getEnvAsInt("MAX_RETRIES", 3),
		RetryDelay:     getEnvAsDuration("RETRY_DELAY", 250*time.Millisecond),

		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://admin:changeme@localhost:5672"),

		FileAgeThreshold:   getEnvAsDuration("FILE_AGE_THRESHOLD", 30*time.Second),
		FileProcessTimeout: getEnvAsDuration("FILE_PROCESS_TIMEOUT", 10*time.Minute),

		GoMaxProcs: getEnvAsInt("GOMAXPROCS", 20),

		// Development & Monitoring
		EnablePprof:  getEnvAsBool("ENABLE_PPROF", false),
		PprofPort:    getEnv("PPROF_PORT", "6060"),
		MemoryTuning: getEnvAsBool("MEMORY_TUNING", true),

		RabbitMQPoolSize:      getEnvAsInt("RABBITMQ_POOL_SIZE", 20),                          // Match worker count
		RabbitMQPrefetchCount: getEnvAsInt("RABBITMQ_PREFETCH_COUNT", 100000),                 // Large prefetch buffer
		RabbitMQBatchSize:     getEnvAsInt("RABBITMQ_BATCH_SIZE", 16000),                      // Large message batches
		RabbitMQBatchTimeout:  getEnvAsDuration("RABBITMQ_BATCH_TIMEOUT", 2*time.Millisecond), // Minimal delay
		RabbitMQConfirms:      getEnvAsBool("RABBITMQ_CONFIRMS", false),                       // Speed over reliability
		RabbitMQPersistent:    getEnvAsBool("RABBITMQ_PERSISTENT", false),                     // Speed over durability
		RabbitMQHeartbeat:     getEnvAsDuration("RABBITMQ_HEARTBEAT", 60*time.Second),         // Reduce overhead
		RabbitMQChannelMax:    getEnvAsInt("RABBITMQ_CHANNEL_MAX", 8192),                      // High channel capacity
		RabbitMQFrameSize:     getEnvAsInt("RABBITMQ_FRAME_SIZE", 16777216),                   // 4MB frames for large batches

		// Record Processing
		BatchSizeRecords: getEnvAsInt("BATCH_SIZE_RECORDS", 64000),
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
