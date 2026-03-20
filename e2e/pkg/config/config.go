package config

import (
	"time"
)

type E2eTestingConfig struct {
	RecordCount int
	BatchSize   int
	SessionID   string

	publisherWorkers int
	PublisherRate    int

	NetworkLatency   time.Duration
	NetworkBandwidth int // Mbps
	NetworkJitter    time.Duration
	PacketLoss       float32

	verificationTimeout time.Duration
	IntegritySampleSize int

	RabbitMQMemory         string
	QuestDBMemory          string
	TelemetryServiceMemory string
}
