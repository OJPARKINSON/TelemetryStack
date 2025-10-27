package worker

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/messaging"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/metrics"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Helper to convert os.FileInfo to os.DirEntry for retry logic
type dirEntryFromFileInfo struct {
	os.FileInfo
}

func (d *dirEntryFromFileInfo) Type() os.FileMode {
	return d.FileInfo.Mode().Type()
}

func (d *dirEntryFromFileInfo) Info() (os.FileInfo, error) {
	return d.FileInfo, nil
}

type WorkerPool struct {
	config      *config.Config
	fileQueue   chan WorkItem
	resultsChan chan WorkResult
	errorsChan  chan WorkError
	ctx         context.Context
	cancel      context.CancelFunc
	eg          *errgroup.Group
	metrics     PoolMetrics
	mu          sync.Mutex

	rabbitPool *messaging.ConnectionPool
	logger     *zap.Logger

	workerMetrics   []WorkerMetrics
	progressDisplay *ProgressDisplay

	// Data loss monitoring
	totalRabbitMQFailures     int
	totalPersistedBatches     int
	totalCircuitBreakerEvents int
	totalMemoryPressureEvents int
}

type PoolMetrics struct {
	TotalFilesProcessed   int
	TotalRecordsProcessed int
	TotalBatchesProcessed int
	TotalErrors           int
	StartTime             time.Time
	ActiveWorkers         int
	QueueDepth            int
	WorkerMetrics         []WorkerMetrics
	
	// Data loss tracking
	RabbitMQFailures      int
	PersistedBatches      int
	CircuitBreakerEvents  int
	MemoryPressureEvents  int
	DataLossRate          float64 // Percentage of data that was lost vs persisted
}

func NewWorkerPool(cfg *config.Config, logger *zap.Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	var rabbitPool *messaging.ConnectionPool
	var err error

	if !cfg.DisableRabbitMQ {
		rabbitPool, err = messaging.NewConnectionPool(cfg.RabbitMQURL, cfg.RabbitMQPoolSize)
		if err != nil {
			logger.Fatal("Failed to create RabbitMQ connection pool",
				zap.Error(err),
				zap.String("url", cfg.RabbitMQURL),
				zap.String("action", "Verify RabbitMQ is running and accessible"))
		}
	}

	workerMetrics := make([]WorkerMetrics, cfg.WorkerCount)
	for i := range workerMetrics {
		workerMetrics[i] = WorkerMetrics{
			WorkerID:     i,
			LastActivity: time.Now(),
			Status:       "IDLE",
		}
	}

	return &WorkerPool{
		config:        cfg,
		fileQueue:     make(chan WorkItem, cfg.FileQueueSize),
		resultsChan:   make(chan WorkResult, cfg.WorkerCount*2),
		errorsChan:    make(chan WorkError, cfg.WorkerCount*2),
		ctx:           ctx,
		cancel:        cancel,
		rabbitPool:    rabbitPool,
		logger:        logger,
		workerMetrics: workerMetrics,
		metrics: PoolMetrics{
			StartTime:     time.Now(),
			WorkerMetrics: workerMetrics,
		},
	}
}

func (wp *WorkerPool) SetProgressDisplay(pd *ProgressDisplay) {
	wp.progressDisplay = pd
}

func (wp *WorkerPool) Start() error {
	eg, ctx := errgroup.WithContext(wp.ctx)
	wp.eg = eg
	wp.ctx = ctx

	// Start result collector
	wp.eg.Go(func() error {
		wp.resultCollector()
		return nil
	})

	// Start error collector
	wp.eg.Go(func() error {
		wp.errorCollector()
		return nil
	})

	// Start workers
	for i := 0; i < wp.config.WorkerCount; i++ {
		workerID := i
		wp.eg.Go(func() error {
			wp.startWorker(workerID)
			return nil
		})
	}

	wp.mu.Lock()
	wp.metrics.ActiveWorkers = wp.config.WorkerCount
	wp.mu.Unlock()

	// Update Prometheus metrics
	metrics.ActiveWorkers.Set(float64(wp.config.WorkerCount))

	return nil
}

func (wp *WorkerPool) SubmitFile(item WorkItem) error {
	select {
	case wp.fileQueue <- item:
		wp.mu.Lock()
		wp.metrics.QueueDepth++
		metrics.QueueDepth.Set(float64(wp.metrics.QueueDepth))
		wp.mu.Unlock()
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	}
}

func (wp *WorkerPool) Stop() error {
	// First close the file queue to prevent new work
	close(wp.fileQueue)

	// Give workers a chance to finish current files and flush data
	time.Sleep(2 * time.Second)

	// Now cancel the context for any remaining operations
	wp.cancel()

	// Wait for all workers to finish using errgroup
	err := wp.eg.Wait()

	if wp.rabbitPool != nil {
		wp.rabbitPool.Close()
	}

	close(wp.resultsChan)
	close(wp.errorsChan)

	// Stop progress display
	if wp.progressDisplay != nil {
		wp.progressDisplay.Stop()
	}

	wp.logFinalMetrics()

	return err
}

func (wp *WorkerPool) GetMetrics() PoolMetrics {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	metrics := wp.metrics
	metrics.QueueDepth = len(wp.fileQueue)

	metrics.WorkerMetrics = make([]WorkerMetrics, len(wp.workerMetrics))
	copy(metrics.WorkerMetrics, wp.workerMetrics)

	// Copy data loss tracking metrics
	metrics.RabbitMQFailures = wp.totalRabbitMQFailures
	metrics.PersistedBatches = wp.totalPersistedBatches
	metrics.CircuitBreakerEvents = wp.totalCircuitBreakerEvents
	metrics.MemoryPressureEvents = wp.totalMemoryPressureEvents
	
	// Calculate data loss rate
	totalBatches := metrics.TotalBatchesProcessed + wp.totalPersistedBatches
	if totalBatches > 0 {
		// Data loss rate = (persisted batches / total batches) * 100
		// This represents the percentage of data that had to be persisted due to RabbitMQ failures
		metrics.DataLossRate = (float64(wp.totalPersistedBatches) / float64(totalBatches)) * 100
	}

	return metrics
}

func (wp *WorkerPool) UpdateWorkerStatus(workerID int, currentFile, status string) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if workerID >= 0 && workerID < len(wp.workerMetrics) {
		wp.workerMetrics[workerID].CurrentFile = currentFile
		wp.workerMetrics[workerID].Status = status
		wp.workerMetrics[workerID].LastActivity = time.Now()

		if wp.progressDisplay != nil {
			wp.progressDisplay.UpdateWorker(workerID, currentFile, status)
		}
	}
}

func (wp *WorkerPool) resultCollector() {
	for {
		select {
		case result, ok := <-wp.resultsChan:
			if !ok {
				return
			}
			wp.handleResult(result)
		case <-wp.ctx.Done():
			return
		}
	}
}

func (wp *WorkerPool) errorCollector() {
	for {
		select {
		case workError, ok := <-wp.errorsChan:
			if !ok {
				return
			}
			wp.handleError(workError)
		case <-wp.ctx.Done():
			return
		}
	}
}

func (wp *WorkerPool) handleResult(result WorkResult) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	wp.metrics.TotalFilesProcessed++
	wp.metrics.TotalRecordsProcessed += result.ProcessedCount
	wp.metrics.TotalBatchesProcessed += result.BatchCount
	wp.metrics.QueueDepth--

	// Update Prometheus metrics
	metrics.FilesProcessedTotal.Inc()
	metrics.RecordsProcessedTotal.Add(float64(result.ProcessedCount))
	metrics.BatchesSentTotal.Add(float64(result.BatchCount))
	metrics.FileProcessingDuration.Observe(result.Duration.Seconds())
	metrics.QueueDepth.Set(float64(wp.metrics.QueueDepth))
	
	// Aggregate messaging metrics from result if available
	if result.MessagingMetrics != nil {
		wp.totalRabbitMQFailures += result.MessagingMetrics.FailedBatches
		wp.totalPersistedBatches += result.MessagingMetrics.PersistedBatches
		if result.MessagingMetrics.CircuitBreakerOpen {
			wp.totalCircuitBreakerEvents++
		}
	}

	if result.WorkerID >= 0 && result.WorkerID < len(wp.workerMetrics) {
		wm := &wp.workerMetrics[result.WorkerID]
		wm.FilesProcessed++
		wm.TotalRecords += int64(result.ProcessedCount)
		wm.TotalBatches += int64(result.BatchCount)
		wm.TotalFileTime += result.Duration
		wm.LastActivity = time.Now()
		wm.CurrentFile = ""
		wm.Status = "IDLE"

		if wm.FilesProcessed > 0 {
			wm.AvgTimePerFile = wm.TotalFileTime / time.Duration(wm.FilesProcessed)
		}

		if result.Duration.Seconds() > 0 {
			wm.ProcessingRate = float64(result.ProcessedCount) / result.Duration.Seconds()
		}
	}

	if wp.progressDisplay != nil {
		if result.WorkerID >= 0 && result.WorkerID < len(wp.workerMetrics) {
			wm := &wp.workerMetrics[result.WorkerID]

			// Update status to IDLE (worker is now waiting for more work)
			wp.progressDisplay.UpdateWorker(result.WorkerID, "", "IDLE")

			// Update stats
			throughput := 0.0
			if result.Duration.Seconds() > 0 {
				// Rough estimate: assuming average record size of 512 bytes
				bytesProcessed := float64(result.ProcessedCount * 512)
				throughput = (bytesProcessed / 1024 / 1024) / result.Duration.Seconds()
			}

			wp.progressDisplay.UpdateWorkerStats(result.WorkerID, WorkerStats{
				FilesProcessed:   wm.FilesProcessed,
				RecordsProcessed: int(wm.TotalRecords),
				Throughput:       throughput,
			})
		}
	}

	// Removed non-actionable Info log - metrics tracked via Prometheus
}

func (wp *WorkerPool) handleError(workError WorkError) {
	wp.mu.Lock()
	wp.metrics.TotalErrors++
	wp.metrics.QueueDepth--
	wp.mu.Unlock()

	if workError.Retry && workError.RetryCount < wp.config.MaxRetries {
		// Try to get FileInfo for retry
		fileInfo, err := os.Stat(workError.FilePath)
		if err != nil {
			wp.logger.Error("Cannot retry file",
				zap.String("file_path", workError.FilePath),
				zap.Error(err),
				zap.String("action", "Check file exists and has read permissions"))
			return
		}

		// Convert to DirEntry for compatibility
		dirEntry := &dirEntryFromFileInfo{fileInfo}

		retryItem := WorkItem{
			FilePath:   workError.FilePath,
			FileInfo:   dirEntry,
			RetryCount: workError.RetryCount + 1,
		}

		time.AfterFunc(wp.config.RetryDelay, func() {
			select {
			case wp.fileQueue <- retryItem:
				// Retry scheduled - no log needed
			case <-wp.ctx.Done():
			}
		})
	} else {
		wp.logger.Error("File processing failed",
			zap.String("file_path", workError.FilePath),
			zap.Int("attempts", workError.RetryCount+1),
			zap.Error(workError.Error),
			zap.String("action", "Check file format is valid IBT or investigate error above"))
	}
}

func (wp *WorkerPool) logFinalMetrics() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	// Only log actionable warnings/errors - all metrics available via Prometheus
	totalBatches := wp.metrics.TotalBatchesProcessed + wp.totalPersistedBatches
	if totalBatches > 0 {
		dataLossRate := (float64(wp.totalPersistedBatches) / float64(totalBatches)) * 100

		if dataLossRate > 5.0 {
			wp.logger.Warn("High data persistence rate detected",
				zap.Float64("rate_percent", dataLossRate),
				zap.Int("persisted_batches", wp.totalPersistedBatches),
				zap.Int("total_batches", totalBatches),
				zap.String("action", "Check RabbitMQ connectivity and service health at "+wp.config.RabbitMQURL))
		}
	}

	if wp.metrics.TotalErrors > 0 {
		wp.logger.Error("Processing completed with errors",
			zap.Int("total_errors", wp.metrics.TotalErrors),
			zap.Int("files_processed", wp.metrics.TotalFilesProcessed),
			zap.String("action", "Review error logs above for failed files"))
	}
}
