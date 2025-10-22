package worker

import (
	"fmt"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
)

type ProgressDisplay struct {
	workerCount   int
	expectedFiles int
	workers       map[int]*WorkerStatus
	trackers      map[int]*progress.Tracker
	pw            progress.Writer
	mu            sync.RWMutex
	startTime     time.Time
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

type WorkerStatus struct {
	ID               int
	CurrentFile      string
	Status           string // "IDLE", "PROCESSING", "ERROR"
	FilesProcessed   int
	RecordsProcessed int
	Throughput       float64 // MB/s
}

func NewProgressDisplay(workerCount, expectedFiles int) *ProgressDisplay {
	workers := make(map[int]*WorkerStatus)
	trackers := make(map[int]*progress.Tracker)

	for i := 0; i < workerCount; i++ {
		workers[i] = &WorkerStatus{
			ID:     i,
			Status: "IDLE",
		}
	}

	pw := progress.NewWriter()
	pw.SetAutoStop(false)
	pw.SetTrackerLength(25)
	pw.SetMessageLength(40)
	pw.SetNumTrackersExpected(workerCount)
	pw.SetSortBy(progress.SortByPercentDsc)
	pw.SetStyle(progress.StyleDefault)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 500)
	pw.Style().Colors = progress.StyleColorsExample
	pw.Style().Options.PercentFormat = "%4.1f%%"
	pw.Style().Visibility.ETA = false
	pw.Style().Visibility.ETAOverall = false
	pw.Style().Visibility.Time = false
	pw.Style().Visibility.TrackerOverall = false
	pw.Style().Visibility.Value = true
	pw.Style().Visibility.Percentage = true

	return &ProgressDisplay{
		workerCount:   workerCount,
		expectedFiles: expectedFiles,
		workers:       workers,
		trackers:      trackers,
		pw:            pw,
		startTime:     time.Now(),
	}
}

func (pd *ProgressDisplay) Start() {
	// Initialize trackers for each worker
	pd.mu.Lock()
	for i := 0; i < pd.workerCount; i++ {
		tracker := &progress.Tracker{
			Message: fmt.Sprintf("Worker %d: Idle", i),
			Total:   int64(pd.expectedFiles),
			Units:   progress.UnitsDefault,
		}
		pd.trackers[i] = tracker
		pd.pw.AppendTracker(tracker)
	}
	pd.mu.Unlock()

	go pd.pw.Render()
}

func (pd *ProgressDisplay) Render() {
	// Progress bars auto-render, but we can use this for custom rendering if needed
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	// Update tracker messages with current stats
	for i := 0; i < pd.workerCount; i++ {
		w := pd.workers[i]
		tracker := pd.trackers[i]

		if tracker != nil {
			// Update progress value
			tracker.SetValue(int64(w.FilesProcessed))

			// Build status message
			msg := fmt.Sprintf("Worker %d [%s]: ", i, w.Status)

			if w.CurrentFile != "" {
				// Truncate filename
				filename := w.CurrentFile
				if len(filename) > 25 {
					filename = "..." + filename[len(filename)-22:]
				}
				msg += filename
			} else {
				msg += "Idle"
			}

			// Add stats
			msg += fmt.Sprintf(" | %d files | %d records | %.2f MB/s",
				w.FilesProcessed,
				w.RecordsProcessed,
				w.Throughput)

			tracker.UpdateMessage(msg)
		}
	}
}

func (pd *ProgressDisplay) UpdateWorker(workerID int, filename, status string) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if w, ok := pd.workers[workerID]; ok {
		w.CurrentFile = filename
		w.Status = status
	}

	// Trigger render update
	pd.updateTrackerMessage(workerID)
}

func (pd *ProgressDisplay) UpdateWorkerStats(workerID int, stats WorkerStats) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if w, ok := pd.workers[workerID]; ok {
		w.FilesProcessed = stats.FilesProcessed
		w.RecordsProcessed = stats.RecordsProcessed
		w.Throughput = stats.Throughput
	}

	// Update progress bar
	if tracker, ok := pd.trackers[workerID]; ok {
		tracker.SetValue(int64(stats.FilesProcessed))
	}

	// Trigger render update
	pd.updateTrackerMessage(workerID)
}

func (pd *ProgressDisplay) updateTrackerMessage(workerID int) {
	w := pd.workers[workerID]
	tracker := pd.trackers[workerID]

	if tracker == nil {
		return
	}

	// Build status message with fixed-width formatting
	statusColor := ""
	switch w.Status {
	case "PROCESSING":
		statusColor = "PROC"
	case "COMPLETED":
		statusColor = "DONE"
	case "ERROR":
		statusColor = "ERR "
	case "IDLE":
		statusColor = "IDLE"
	default:
		statusColor = "    "
	}

	// Truncate filename for display
	filename := "waiting..."
	if w.CurrentFile != "" {
		filename = w.CurrentFile
		if len(filename) > 30 {
			filename = "..." + filename[len(filename)-27:]
		}
	}

	// Format: Worker N [STATUS] filename | stats
	msg := fmt.Sprintf("W-%02d [%s] %-30s | Files:%2d Recs:%7d MB/s:%6.2f",
		workerID,
		statusColor,
		filename,
		w.FilesProcessed,
		w.RecordsProcessed,
		w.Throughput)

	tracker.UpdateMessage(msg)
}

func (pd *ProgressDisplay) Stop() {
	// Mark all trackers as done
	pd.mu.Lock()
	for _, tracker := range pd.trackers {
		if tracker != nil {
			tracker.MarkAsDone()
		}
	}
	pd.mu.Unlock()

	// Wait briefly for final render with timeout
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// Timeout reached, force stop
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

type WorkerStats struct {
	FilesProcessed   int
	RecordsProcessed int
	Throughput       float64
}
