package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// File processing metrics
	FilesProcessedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ingest_files_processed_total",
		Help: "Total number of IBT files processed",
	})

	FileProcessingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ingest_file_processing_duration_seconds",
		Help:    "Time taken to process a single IBT file",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~102s
	})

	// Record throughput metrics
	RecordsProcessedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ingest_records_processed_total",
		Help: "Total number of telemetry records processed",
	})

	// Batch metrics
	BatchesSentTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ingest_batches_sent_total",
		Help: "Total number of batches sent to the telemetry service",
	})

	BatchSizeBytes = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ingest_batch_size_bytes",
		Help:    "Size of batches put on the wire, in bytes (post-compression)",
		Buckets: prometheus.ExponentialBuckets(1024*1024, 2, 8), // 1MB to 128MB
	})

	BatchBytesUncompressed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ingest_batch_bytes_uncompressed_total",
		Help: "Total marshalled batch bytes before compression",
	})

	BatchBytesWire = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ingest_batch_bytes_wire_total",
		Help: "Total batch bytes actually sent over the network",
	})

	// Worker pool metrics
	ActiveWorkers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ingest_active_workers",
		Help: "Current number of active workers",
	})

	QueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ingest_queue_depth",
		Help: "Current depth of the file processing queue",
	})
)
