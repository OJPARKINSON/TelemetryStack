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
	wp.logger.Info("Worker starting", zap.Int("worker_id", workerID))

	workerCtx, cancel := context.WithTimeout(wp.ctx, wp.config.WorkerTimeout)
	defer cancel()

	for {
		select {
		case workItem, ok := <-wp.fileQueue:
			if !ok {
				wp.logger.Info("Worker shutting down - queue closed", zap.Int("worker_id", workerID))
				return
			}

			wp.processWorkItem(workerCtx, workerID, workItem)

		case <-workerCtx.Done():
			wp.logger.Info("Worker shutting down - context done", zap.Int("worker_id", workerID), zap.Error(workerCtx.Err()))
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

	wp.logger.Info("Worker processing file", zap.Int("worker_id", workerID), zap.String("file_path", item.FilePath), zap.Int("retry", item.RetryCount))

	processor, err := processing.NewFileProcessor(wp.config, workerID, wp.rabbitPool)

	if err != nil {
		wp.logger.Error("Worker error creating file processor", zap.Int("worker_id", workerID), zap.String("filename", filename), zap.Error(err))
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
			wp.logger.Warn("Worker file processing timeout", zap.Int("worker_id", workerID), zap.String("filename", filename))
		} else {
			wp.logger.Info("Worker graceful shutdown - attempting data flush", zap.Int("worker_id", workerID), zap.String("filename", filename))
			if flushErr := processor.FlushPendingData(); flushErr != nil {
				wp.logger.Error("Worker failed to flush pending data", zap.Int("worker_id", workerID), zap.String("filename", filename), zap.Error(flushErr))
			}
			processErr = fmt.Errorf("processing cancelled due to graceful shutdown")
		}
	}

	if processErr != nil {
		wp.logger.Error("Worker error processing file", zap.Int("worker_id", workerID), zap.String("filename", filename), zap.Error(processErr))
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
