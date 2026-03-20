package processing

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/messaging"
	"github.com/OJPARKINSON/ibt"
)

type FileProcessor struct {
	config           *config.Config
	workerID         int
	pool             *messaging.ConnectionPool
	progressCallback ProgressCallback
}

type ProcessResult struct {
	RecordCount      int
	BatchCount       int
	SessionID        string
	TrackName        string
	MessagingMetrics *messaging.PublishMetrics
}

func NewFileProcessor(cfg *config.Config, workerID int, pool *messaging.ConnectionPool) (*FileProcessor, error) {
	return &FileProcessor{
		config:           cfg,
		workerID:         workerID,
		pool:             pool,
		progressCallback: &NoOpProgressCallback{},
	}, nil
}

func (fp *FileProcessor) SetProgressCallback(callback ProgressCallback) {
	if callback != nil {
		fp.progressCallback = callback
	}
}

func (fp *FileProcessor) ProcessFile(ctx context.Context, telemetryFolder string, fileEntry os.DirEntry) (*ProcessResult, error) {
	fileName := fileEntry.Name()

	if !strings.Contains(fileName, ".ibt") {
		return nil, fmt.Errorf("not an IBT file: %s\nAction: Ensure file has .ibt extension", fileName)
	}

	sessionTime, err := fp.parseFileName(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time from filename %s: %w\nAction: Ensure filename contains date pattern YYYY-MM-DD HH-MM-SS", fileName, err)
	}

	file := filepath.Join(telemetryFolder, fileName)

	stubs, err := ibt.ParseStubs(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse stubs for %v: %w\nAction: File may be corrupted or incomplete - verify file integrity", file, err)
	}

	if len(stubs) == 0 {
		return nil, fmt.Errorf("no telemetry data found in IBT file: %s\nAction: File is empty or contains no valid telemetry data", fileName)
	}

	groups := stubs.Group()

	// Pre-count total records for progress tracking
	totalExpectedRecords := 0
	for _, group := range groups {
		totalExpectedRecords += len(group)
	}

	// Notify progress callback that file processing is starting
	fp.progressCallback.OnFileStart(fileName, totalExpectedRecords)

	totalRecords := 0
	totalBatches := 0

	processors := make([]ibt.Processor, 0, len(groups))

	// Track metrics across all groups
	var allMessagingMetrics *messaging.PublishMetrics
	var firstSessionID string
	var firstTrackName string

	for groupNumber, group := range groups {
		select {
		case <-ctx.Done():
			// Graceful shutdown: flush all active processors
			log.Printf("Context cancelled, flushing %d active processors", len(processors))
			for i, proc := range processors {
				if flushErr := proc.FlushPendingData(); flushErr != nil {
					log.Printf("Failed to flush processor %d: %v", i, flushErr)
				}
			}
			return nil, ctx.Err()
		default:
		}

		// Extract SubSessionID from this group
		if len(group) == 0 {
			continue
		}

		groupHeaders := group[0].Headers()
		groupWeekendInfo := groupHeaders.SessionInfo.WeekendInfo
		groupSessionID := strconv.Itoa(groupWeekendInfo.SubSessionID)

		// Track first session's info for result
		if groupNumber == 0 {
			firstSessionID = groupSessionID
			firstTrackName = groupWeekendInfo.TrackDisplayName
		}

		// Create PubSub for this specific group
		pubSub := messaging.NewPubSub(
			groupSessionID,
			sessionTime,
			fp.config,
			fp.pool,
			fp.workerID,
		)

		// Create telemetry processor with the correct SubSessionID
		processor := NewProcessor(pubSub, groupNumber, fp.config, fp.workerID, groupSessionID)
		processor.SetProgressCallback(fp.progressCallback, fileName)
		processors = append(processors, processor)

		if err := ibt.Process(ctx, group, processor); err != nil {
			// Try to flush this processor before returning error
			if flushErr := processor.FlushPendingData(); flushErr != nil {
				log.Printf("Failed to flush processor on error: %v", flushErr)
			}
			pubSub.Close()
			return nil, err
		}

		if err := processor.Close(); err != nil {
			pubSub.Close()
			return nil, fmt.Errorf("error closing processor for group %d: %w\nAction: Check RabbitMQ connectivity and disk space", groupNumber, err)
		}

		// Collect metrics from this group's PubSub
		metrics := pubSub.GetMetrics()
		if allMessagingMetrics == nil {
			allMessagingMetrics = &metrics
		} else {
			// Accumulate metrics across groups
			allMessagingMetrics.TotalBatches += metrics.TotalBatches
			allMessagingMetrics.TotalRecords += metrics.TotalRecords
			allMessagingMetrics.TotalBytes += metrics.TotalBytes
			allMessagingMetrics.FailedBatches += metrics.FailedBatches
			allMessagingMetrics.PersistedBatches += metrics.PersistedBatches
		}

		// Close PubSub for this group
		if err := pubSub.Close(); err != nil {
			log.Printf("Failed to close PubSub for group %d: %v", groupNumber, err)
		}
	}

	ibt.CloseAllStubs(groups)

	// Notify progress callback that file processing is complete
	fp.progressCallback.OnFileComplete(fileName)

	return &ProcessResult{
		RecordCount:      totalRecords,
		BatchCount:       totalBatches,
		SessionID:        firstSessionID,
		TrackName:        firstTrackName,
		MessagingMetrics: allMessagingMetrics,
	}, nil
}

func (fp *FileProcessor) FlushPendingData() error {
	// No-op: each processor handles its own flushing
	return nil
}

func (fp *FileProcessor) Close() error {
	// No-op: PubSub instances are closed per-group
	return nil
}

func (fp *FileProcessor) parseFileName(fileName string) (time.Time, error) {
	regex := regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}-\d{2}-\d{2}`)

	match := regex.FindString(fileName)
	if match == "" {
		return time.Time{}, fmt.Errorf("no date pattern found in filename: %s\nAction: Filename must contain YYYY-MM-DD HH-MM-SS pattern", fileName)
	}

	parts := strings.Split(match, " ")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid date format: %s\nAction: Date pattern must be YYYY-MM-DD HH-MM-SS", match)
	}

	timeStr := strings.ReplaceAll(parts[1], "-", ":")
	rfc3339Str := parts[0] + "T" + timeStr + "Z"

	parsedTime, err := time.Parse(time.RFC3339, rfc3339Str)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse time %s: %w\nAction: Verify date format is valid YYYY-MM-DD HH-MM-SS", rfc3339Str, err)
	}

	return parsedTime, nil
}
