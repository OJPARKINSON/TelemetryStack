package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Records received from HTTP ingest
	RecordsReceivedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "telemetry_records_received_total",
		Help: "Total number of telemetry records received from ingest",
	})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "telemetry_http_request_duration_seconds",
		Help:    "HTTP request duration by endpoint, method, and status code",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 14), // 1ms to ~16s
	}, []string{"path", "method", "status"})

	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "telemetry_http_requests_total",
		Help: "Total HTTP requests by endpoint, method, and status code",
	}, []string{"path", "method", "status"})

	HTTPRequestsInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "telemetry_http_requests_in_flight",
		Help: "Number of HTTP requests currently being served",
	})

	QuestDBQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "telemetry_questdb_query_duration_seconds",
		Help:    "QuestDB read query duration by operation",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 14),
	}, []string{"operation"})

	QuestDBQueryErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "telemetry_questdb_query_errors_total",
		Help: "Total QuestDB read query errors by operation",
	}, []string{"operation"})

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

	// Queue metrics
	QueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "telemetry_queue_depth",
		Help: "Current number of batches in the ingest queue",
	})

	QueueEnqueueDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "telemetry_queue_enqueue_duration_seconds",
		Help:    "Time spent waiting to enqueue a batch (backpressure indicator)",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 14), // 1ms to ~16s
	})

	QueueBatchesFlushed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "telemetry_queue_batches_flushed_total",
		Help: "Total batches flushed by queue workers",
	})

	QueueCoalescedBatches = promauto.NewCounter(prometheus.CounterOpts{
		Name: "telemetry_queue_coalesced_batches_total",
		Help: "Total extra batches merged during drain coalescing",
	})
)
