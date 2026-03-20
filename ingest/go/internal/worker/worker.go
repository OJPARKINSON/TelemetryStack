package worker

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/processing"
	"go.uber.org/zap"
)

func (wp *WorkerPool) startWorker(workerID int) {
	workerCtx, cancel := context.WithTimeout(wp.ctx, wp.config.WorkerTimeout)
	defer cancel()

	for {
		select {
		case workItem, ok := <-wp.fileQueue:
			if !ok {
				// Worker shutting down - queue closed (normal shutdown)
				return
			}

			wp.processWorkItem(workerCtx, workerID, workItem)

		case <-workerCtx.Done():
			// Worker shutting down due to context cancellation (normal shutdown)
			return
		}
	}
}

func (wp *WorkerPool) processWorkItem(ctx context.Context, workerID int, item WorkItem) {
	startTime := time.Now()

	if item.FileInfo == nil {
		wp.logger.Error("Worker FileInfo is nil", zap.Int("worker_id", workerID), zap.String("file_path", item.FilePath))
		wp.errorsChan <- WorkError{
			FilePath:   item.FilePath,
			Error:      fmt.Errorf("FileInfo is nil"),
			Retry:      false,
			WorkerID:   workerID,
			RetryCount: item.RetryCount,
			Timestamp:  time.Now(),
		}
		return
	}

	filename := item.FileInfo.Name()
	wp.UpdateWorkerStatus(workerID, filename, "PROCESSING")

	processor, err := processing.NewFileProcessor(wp.config, workerID, wp.rabbitPool)

	if err != nil {
		wp.logger.Error("Failed to create file processor",
			zap.String("file", item.FilePath),
			zap.Error(err),
			zap.String("action", "Check system resources and RabbitMQ connectivity"))
		wp.UpdateWorkerStatus(workerID, filename, "ERROR")
		wp.errorsChan <- WorkError{
			FilePath:   item.FilePath,
			Error:      err,
			Retry:      true,
			WorkerID:   workerID,
			RetryCount: item.RetryCount,
			Timestamp:  time.Now(),
		}
		return
	}
	defer processor.Close()

	// Set progress callback if progress display is available
	if wp.progressDisplay != nil {
		processor.SetProgressCallback(wp.progressDisplay)
	}

	processCtx, processCancel := context.WithTimeout(ctx, wp.config.FileProcessTimeout)
	defer processCancel()

	resultChan := make(chan struct {
		result *processing.ProcessResult
		err    error
	}, 1)

	go func() {
		result, err := processor.ProcessFile(processCtx, filepath.Dir(item.FilePath), item.FileInfo)
		resultChan <- struct {
			result *processing.ProcessResult
			err    error
		}{result, err}
	}()

	// Wait for result or timeout
	var result *processing.ProcessResult
	var processErr error
	select {
	case res := <-resultChan:
		result = res.result
		processErr = res.err
	case <-processCtx.Done():
		if processCtx.Err() == context.DeadlineExceeded {
			processErr = fmt.Errorf("file processing timeout after %v", wp.config.FileProcessTimeout)
			wp.logger.Error("File processing timeout",
				zap.String("file", item.FilePath),
				zap.Duration("timeout", wp.config.FileProcessTimeout),
				zap.String("action", "File may be corrupted or unusually large - consider increasing FILE_PROCESS_TIMEOUT"))
		} else {
			// Graceful shutdown - attempt data flush
			if flushErr := processor.FlushPendingData(); flushErr != nil {
				wp.logger.Error("Failed to flush pending data during shutdown",
					zap.String("file", item.FilePath),
					zap.Error(flushErr),
					zap.String("action", "Some data may be lost - check disk persistence directory"))
			}
			processErr = fmt.Errorf("processing cancelled due to graceful shutdown")
		}
	}

	if processErr != nil {
		wp.UpdateWorkerStatus(workerID, filename, "ERROR")
		wp.errorsChan <- WorkError{
			FilePath:   item.FilePath,
			Error:      processErr,
			Retry:      shouldRetry(processErr, item.RetryCount),
			WorkerID:   workerID,
			RetryCount: item.RetryCount,
			Timestamp:  time.Now(),
		}
		return
	}

	wp.resultsChan <- WorkResult{
		FilePath:         item.FilePath,
		ProcessedCount:   result.RecordCount,
		BatchCount:       result.BatchCount,
		Duration:         time.Since(startTime),
		SessionID:        result.SessionID,
		TrackName:        result.TrackName,
		WorkerID:         workerID,
		MessagingMetrics: result.MessagingMetrics,
	}
}

func shouldRetry(err error, retryCount int) bool {
	if retryCount >= 3 {
		return false
	}

	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	return true
}
