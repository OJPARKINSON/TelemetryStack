package worker

import (
	"fmt"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
)

// FileProgress tracks progress for a single file
type FileProgress struct {
	Filename      string
	TotalRecords  int
	RecordsSent   int
	BatchesSent   int
	WorkerID      int
	StartTime     time.Time
	Tracker       *progress.Tracker
}

// ProgressDisplay shows real-time progress for file processing
type ProgressDisplay struct {
	files         map[string]*FileProgress // filename -> progress
	pw            progress.Writer
	mu            sync.RWMutex
	startTime     time.Time
	totalFiles    int
	completedFiles int
	overallTracker *progress.Tracker
}

func NewProgressDisplay(workerCount, expectedFiles int) *ProgressDisplay {
	pw := progress.NewWriter()
	pw.SetAutoStop(false)
	pw.SetTrackerLength(30)
	pw.SetMessageLength(60)
	pw.SetNumTrackersExpected(expectedFiles + 1) // +1 for overall tracker
	pw.SetSortBy(progress.SortByPercentDsc)
	pw.SetStyle(progress.StyleDefault)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 300)
	pw.Style().Colors = progress.StyleColorsExample
	pw.Style().Options.PercentFormat = "%4.1f%%"
	pw.Style().Visibility.ETA = true
	pw.Style().Visibility.ETAOverall = false
	pw.Style().Visibility.Time = false
	pw.Style().Visibility.TrackerOverall = true
	pw.Style().Visibility.Value = true
	pw.Style().Visibility.Percentage = true

	return &ProgressDisplay{
		files:      make(map[string]*FileProgress),
		pw:         pw,
		startTime:  time.Now(),
		totalFiles: expectedFiles,
	}
}

func (pd *ProgressDisplay) Start() {
	// Create overall progress tracker
	pd.mu.Lock()
	pd.overallTracker = &progress.Tracker{
		Message: "Overall Progress",
		Total:   int64(pd.totalFiles),
		Units:   progress.UnitsDefault,
	}
	pd.pw.AppendTracker(pd.overallTracker)
	pd.mu.Unlock()

	go pd.pw.Render()
}

// OnFileStart implements ProgressCallback interface
func (pd *ProgressDisplay) OnFileStart(filename string, totalRecords int) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	// Truncate filename for display
	displayName := filename
	if len(displayName) > 35 {
		displayName = "..." + displayName[len(displayName)-32:]
	}

	fileProgress := &FileProgress{
		Filename:     filename,
		TotalRecords: totalRecords,
		RecordsSent:  0,
		BatchesSent:  0,
		StartTime:    time.Now(),
	}

	tracker := &progress.Tracker{
		Message: fmt.Sprintf("%-35s | W-?? | Batches: 0", displayName),
		Total:   int64(totalRecords),
		Units:   progress.UnitsDefault,
	}

	fileProgress.Tracker = tracker
	pd.files[filename] = fileProgress
	pd.pw.AppendTracker(tracker)
}

// OnBatchSent implements ProgressCallback interface
func (pd *ProgressDisplay) OnBatchSent(filename string, recordsSent int, batchNum int) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	fileProgress, exists := pd.files[filename]
	if !exists {
		return
	}

	fileProgress.RecordsSent = recordsSent
	fileProgress.BatchesSent = batchNum

	// Update tracker value and message
	if fileProgress.Tracker != nil {
		fileProgress.Tracker.SetValue(int64(recordsSent))

		// Truncate filename for display
		displayName := filename
		if len(displayName) > 35 {
			displayName = "..." + displayName[len(displayName)-32:]
		}

		// Calculate throughput
		elapsed := time.Since(fileProgress.StartTime).Seconds()
		recordsPerSec := 0.0
		if elapsed > 0 {
			recordsPerSec = float64(recordsSent) / elapsed
		}

		msg := fmt.Sprintf("%-35s | W-%02d | Batches: %3d | %7.0f rec/s",
			displayName,
			fileProgress.WorkerID,
			batchNum,
			recordsPerSec)

		fileProgress.Tracker.UpdateMessage(msg)
	}
}

// OnFileComplete implements ProgressCallback interface
func (pd *ProgressDisplay) OnFileComplete(filename string) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	fileProgress, exists := pd.files[filename]
	if !exists {
		return
	}

	// Mark tracker as done
	if fileProgress.Tracker != nil {
		fileProgress.Tracker.MarkAsDone()
	}

	// Update overall progress
	pd.completedFiles++
	if pd.overallTracker != nil {
		pd.overallTracker.SetValue(int64(pd.completedFiles))

		elapsed := time.Since(pd.startTime)
		avgTimePerFile := elapsed / time.Duration(pd.completedFiles)
		remaining := time.Duration(pd.totalFiles-pd.completedFiles) * avgTimePerFile

		msg := fmt.Sprintf("Overall Progress | %d/%d files | Elapsed: %v | ETA: %v",
			pd.completedFiles,
			pd.totalFiles,
			elapsed.Round(time.Second),
			remaining.Round(time.Second))

		pd.overallTracker.UpdateMessage(msg)
	}
}

// UpdateWorker is called from the worker pool to associate files with workers
func (pd *ProgressDisplay) UpdateWorker(workerID int, filename, status string) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if filename == "" {
		return
	}

	fileProgress, exists := pd.files[filename]
	if exists {
		fileProgress.WorkerID = workerID
	}
}

// UpdateWorkerStats is called from the worker pool (legacy compatibility)
func (pd *ProgressDisplay) UpdateWorkerStats(workerID int, stats WorkerStats) {
	// This method is kept for compatibility but not actively used
	// File progress is now tracked via ProgressCallback interface
}

func (pd *ProgressDisplay) Stop() {
	pd.mu.Lock()

	// Mark all remaining trackers as done
	for _, fileProgress := range pd.files {
		if fileProgress.Tracker != nil && !fileProgress.Tracker.IsDone() {
			fileProgress.Tracker.MarkAsDone()
		}
	}

	if pd.overallTracker != nil {
		pd.overallTracker.MarkAsDone()
	}
	pd.mu.Unlock()

	// Wait briefly for final render with timeout
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// Force stop after timeout
			pd.pw.Stop()
			return
		case <-ticker.C:
			if !pd.pw.IsRenderInProgress() {
				pd.pw.Stop()
				return
			}
		}
	}
}

// Legacy types for compatibility
type WorkerStatus struct {
	ID               int
	CurrentFile      string
	Status           string
	FilesProcessed   int
	RecordsProcessed int
	Throughput       float64
}

type WorkerStats struct {
	FilesProcessed   int
	RecordsProcessed int
	Throughput       float64
}

type WorkerProgress struct {
	ID               int
	FilesProcessed   int
	RecordsProcessed int64
	BatchesProcessed int64
	CurrentFile      string
	Status           WorkerStatus
	LastActivity     time.Time
	ProcessingRate   float64
	AvgTimePerFile   time.Duration
	TotalFileTime    time.Duration
}
