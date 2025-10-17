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
	config   *config.Config
	workerID int
	pubSub   *messaging.PubSub
	pool     *messaging.ConnectionPool
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
		config:   cfg,
		workerID: workerID,
		pool:     pool,
	}, nil
}

func (fp *FileProcessor) ProcessFile(ctx context.Context, telemetryFolder string, fileEntry os.DirEntry) (*ProcessResult, error) {
	fileName := fileEntry.Name()

	if !strings.Contains(fileName, ".ibt") {
		return nil, fmt.Errorf("not an IBT file: %s", fileName)
	}

	sessionTime, err := fp.parseFileName(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time from filename: %w", err)
	}

	file := filepath.Join(telemetryFolder, fileName)

	// Reduced logging for performance
	stubs, err := ibt.ParseStubs(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse stubs for %v: %w", file, err)
	}

	if len(stubs) == 0 {
		return nil, fmt.Errorf("no telemetry data found in IBT file: %s", fileName)
	}

	headers := stubs[0].Headers()
	weekendInfo := headers.SessionInfo.WeekendInfo

	fp.pubSub = messaging.NewPubSub(
		strconv.Itoa(weekendInfo.SubSessionID),
		sessionTime,
		fp.config,
		fp.pool,
	)

	groups := stubs.Group()

	totalRecords := 0
	totalBatches := 0

	processors := make([]ibt.Processor, 0, len(groups))

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

		var processor ibt.Processor
		if fp.config.UseStructPipeline {
			processor = NewStructProcessor(fp.pubSub, groupNumber, fp.config, fp.workerID)
		} else {
			processor = NewLoaderProcessor(fp.pubSub, groupNumber, fp.config, fp.workerID)
		}
		processors = append(processors, processor)

		if err := ibt.Process(ctx, group, processor); err != nil {
			// Try to flush this processor before returning error
			if flushErr := processor.FlushPendingData(); flushErr != nil {
				log.Printf("Failed to flush processor on error: %v", flushErr)
			}
			return nil, err
		}

		if err := processor.Close(); err != nil {
			return nil, fmt.Errorf("error closing processor for group %d: %w", groupNumber, err)
		}

		metrics := processor.GetMetrics()
		if m, ok := metrics.(ProcessorMetrics); ok {
			totalRecords += m.TotalProcessed
			totalBatches += m.TotalBatches
		}
	}

	ibt.CloseAllStubs(groups)

	// Collect messaging metrics if pubSub exists
	var messagingMetrics *messaging.PublishMetrics
	if fp.pubSub != nil {
		metrics := fp.pubSub.GetMetrics()
		messagingMetrics = &metrics
	}

	return &ProcessResult{
		RecordCount:      totalRecords,
		BatchCount:       totalBatches,
		SessionID:        strconv.Itoa(weekendInfo.SubSessionID),
		TrackName:        weekendInfo.TrackDisplayName,
		MessagingMetrics: messagingMetrics,
	}, nil
}

func (fp *FileProcessor) FlushPendingData() error {
	if fp.pubSub != nil {
		log.Printf("FileProcessor: Flushing pending data for worker %d", fp.workerID)
		return fp.pubSub.FlushBatch()
	}
	return nil
}

func (fp *FileProcessor) Close() error {
	if fp.pubSub != nil {
		return fp.pubSub.Close()
	}
	return nil
}

func (fp *FileProcessor) parseFileName(fileName string) (time.Time, error) {
	regex := regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}-\d{2}-\d{2}`)

	match := regex.FindString(fileName)
	if match == "" {
		return time.Time{}, fmt.Errorf("no date pattern found in filename: %s", fileName)
	}

	parts := strings.Split(match, " ")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", match)
	}

	timeStr := strings.ReplaceAll(parts[1], "-", ":")
	rfc3339Str := parts[0] + "T" + timeStr + "Z"

	parsedTime, err := time.Parse(time.RFC3339, rfc3339Str)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse time %s: %w", rfc3339Str, err)
	}

	return parsedTime, nil
}
