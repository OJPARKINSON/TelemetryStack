package worker

import (
	"os"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/messaging"
)

type WorkItem struct {
	FilePath   string
	FileInfo   os.DirEntry
	RetryCount int
}

type WorkResult struct {
	FilePath         string
	ProcessedCount   int
	BatchCount       int
	Duration         time.Duration
	SessionID        string
	TrackName        string
	WorkerID         int
	MessagingMetrics *messaging.PublishMetrics
}

type WorkError struct {
	FilePath   string
	Error      error
	Retry      bool
	WorkerID   int
	RetryCount int
	Timestamp  time.Time
}

type WorkerMetrics struct {
	WorkerID         int
	FilesProcessed   int
	TotalRecords     int64
	TotalBatches     int64
	ProcessingTime   time.Duration
	LastActivity     time.Time
	ErrorCount       int
	CurrentFile      string
	Status           string
	ProcessingRate   float64
	AvgTimePerFile   time.Duration
	TotalFileTime    time.Duration
}
