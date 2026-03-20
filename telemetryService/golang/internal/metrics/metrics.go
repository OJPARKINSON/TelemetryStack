package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Records received from RabbitMQ
	RecordsReceivedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "telemetry_records_received_total",
		Help: "Total number of telemetry records received from RabbitMQ",
	})

	// Records written to QuestDB
	RecordsWrittenTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "telemetry_records_written_total",
		Help: "Total number of telemetry records successfully written to QuestDB",
	})

	// Database write latency
	DBWriteDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "telemetry_db_write_duration_seconds",
		Help:    "Time taken to write a batch to QuestDB",
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // 10ms to ~10s
	})

	// Write errors
	DBWriteErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "telemetry_db_write_errors_total",
		Help: "Total number of failed QuestDB write operations",
	})

	// Batch size metrics
	BatchSizeRecords = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "telemetry_batch_size_records",
		Help:    "Number of records per batch written to QuestDB",
		Buckets: prometheus.ExponentialBuckets(100, 2, 10), // 100 to ~102k records
	})
)
