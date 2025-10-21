package worker

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/messaging"
	"go.uber.org/zap"
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
	wg          sync.WaitGroup
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
			logger.Fatal("Failed to create RabbitMQ connection pool", zap.Error(err))
		}
		logger.Info("Created RabbitMQ connection pool", zap.Int("connections", cfg.RabbitMQPoolSize))
	} else {
		logger.Info("RabbitMQ disabled - running in benchmark mode")
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

func (wp *WorkerPool) Start() {
	wp.logger.Info("Starting worker pool", zap.Int("workers", wp.config.WorkerCount))

	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		wp.resultCollector()
	}()

	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		wp.errorCollector()
	}()

	for i := 0; i < wp.config.WorkerCount; i++ {
		wp.wg.Add(1)
		workerID := i
		go func() {
			defer wp.wg.Done()
			wp.startWorker(workerID)
		}()
	}

	wp.mu.Lock()
	wp.metrics.ActiveWorkers = wp.config.WorkerCount
	wp.mu.Unlock()

	wp.logger.Info("Worker pool started successfully")
}

func (wp *WorkerPool) SubmitFile(item WorkItem) error {
	select {
	case wp.fileQueue <- item:
		wp.mu.Lock()
		wp.metrics.QueueDepth++
		wp.mu.Unlock()
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	}
}

func (wp *WorkerPool) Stop() {
	wp.logger.Info("Stopping worker pool")

	// First close the file queue to prevent new work
	close(wp.fileQueue)

	// Give workers a chance to finish current files and flush data
	wp.logger.Info("Waiting for workers to complete current tasks and flush data")
	
	// Wait a short time before cancelling context to allow graceful completion
	time.Sleep(2 * time.Second)

	// Now cancel the context for any remaining operations
	wp.cancel()

	// Wait for all workers to finish
	wp.wg.Wait()

	if wp.rabbitPool != nil {
		wp.rabbitPool.Close()
		wp.logger.Info("Closed RabbitMQ connection pool")
	}

	close(wp.resultsChan)
	close(wp.errorsChan)

	wp.logFinalMetrics()
	wp.logger.Info("Worker pool stopped gracefully")
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
			var displayStatus WorkerStatus
			switch status {
			case "PROCESSING":
				displayStatus = StatusProcessing
			case "ERROR":
				displayStatus = StatusError
			case "COMPLETED":
				displayStatus = StatusCompleted
			default:
				displayStatus = StatusIdle
			}

			wm := wp.workerMetrics[workerID]
			wp.progressDisplay.UpdateWorker(
				workerID,
				wm.FilesProcessed,
				wm.TotalRecords,
				wm.TotalBatches,
				wm.CurrentFile,
				displayStatus,
			)
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
		status := StatusCompleted
		if result.WorkerID >= 0 && result.WorkerID < len(wp.workerMetrics) {
			wm := wp.workerMetrics[result.WorkerID]
			wp.progressDisplay.UpdateWorkerWithTiming(
				result.WorkerID,
				wm.FilesProcessed,
				wm.TotalRecords,
				wm.TotalBatches,
				wm.CurrentFile,
				status,
				wm.AvgTimePerFile,
				wm.TotalFileTime,
			)
		}
	}

	wp.logger.Info("Worker completed file",
		zap.Int("worker_id", result.WorkerID),
		zap.String("file_path", result.FilePath),
		zap.Int("records_processed", result.ProcessedCount),
		zap.Int("batches_processed", result.BatchCount),
		zap.Duration("processing_time", result.Duration))
}

func (wp *WorkerPool) handleError(workError WorkError) {
	wp.logger.Error("Worker pool error",
		zap.Int("worker_id", workError.WorkerID),
		zap.String("file_path", workError.FilePath),
		zap.Error(workError.Error))
	wp.mu.Lock()
	wp.metrics.TotalErrors++
	wp.metrics.QueueDepth--
	wp.mu.Unlock()

	wp.logger.Error("Worker error processing file",
		zap.Int("worker_id", workError.WorkerID),
		zap.String("file_path", workError.FilePath),
		zap.Error(workError.Error))

	if workError.Retry && workError.RetryCount < wp.config.MaxRetries {
		// Try to get FileInfo for retry
		fileInfo, err := os.Stat(workError.FilePath)
		if err != nil {
			wp.logger.Error("Cannot retry file - stat error",
				zap.String("file_path", workError.FilePath),
				zap.Error(err))
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
				wp.logger.Info("Retrying file",
					zap.String("file_path", workError.FilePath),
					zap.Int("attempt", retryItem.RetryCount))
			case <-wp.ctx.Done():
			}
		})
	} else {
		wp.logger.Error("File failed permanently after retries",
			zap.String("file_path", workError.FilePath))
	}
}

func (wp *WorkerPool) logFinalMetrics() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	duration := time.Since(wp.metrics.StartTime)

	wp.logger.Info("=== Final Worker Pool Metrics ===")
	wp.logger.Info("Total processing time", zap.Duration("duration", duration))
	wp.logger.Info("Files processed", zap.Int("files", wp.metrics.TotalFilesProcessed))
	wp.logger.Info("Records processed", zap.Int("records", wp.metrics.TotalRecordsProcessed))
	wp.logger.Info("Batches sent", zap.Int("batches", wp.metrics.TotalBatchesProcessed))
	wp.logger.Info("Errors encountered", zap.Int("errors", wp.metrics.TotalErrors))

	// Data loss and reliability metrics
	wp.logger.Info("=== Data Loss Monitoring ===")
	wp.logger.Info("RabbitMQ failures", zap.Int("failures", wp.totalRabbitMQFailures))
	wp.logger.Info("Batches persisted", zap.Int("persisted", wp.totalPersistedBatches))
	wp.logger.Info("Circuit breaker events", zap.Int("events", wp.totalCircuitBreakerEvents))
	wp.logger.Info("Memory pressure events", zap.Int("events", wp.totalMemoryPressureEvents))
	
	totalBatches := wp.metrics.TotalBatchesProcessed + wp.totalPersistedBatches
	if totalBatches > 0 {
		dataLossRate := (float64(wp.totalPersistedBatches) / float64(totalBatches)) * 100
		successRate := (float64(wp.metrics.TotalBatchesProcessed) / float64(totalBatches)) * 100
		wp.logger.Info("Data persistence rate",
			zap.Float64("rate_percent", dataLossRate),
			zap.Int("persisted_batches", wp.totalPersistedBatches),
			zap.Int("total_batches", totalBatches))
		wp.logger.Info("RabbitMQ success rate", zap.Float64("success_rate_percent", successRate))
		
		if dataLossRate > 5.0 {
			wp.logger.Warn("High data persistence rate detected! Check RabbitMQ connectivity")
		}
	}

	if wp.metrics.TotalFilesProcessed > 0 {
		avgPerFile := duration / time.Duration(wp.metrics.TotalFilesProcessed)
		wp.logger.Info("Average time per file", zap.Duration("avg_time", avgPerFile))
	}

	if duration.Seconds() > 0 {
		filesPerSec := float64(wp.metrics.TotalFilesProcessed) / duration.Seconds()
		recordsPerSec := float64(wp.metrics.TotalRecordsProcessed) / duration.Seconds()
		wp.logger.Info("Throughput",
			zap.Float64("files_per_sec", filesPerSec),
			zap.Float64("records_per_sec", recordsPerSec))
	}
}
