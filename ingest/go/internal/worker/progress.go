package worker

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"golang.org/x/term"
)

type ProgressDisplay struct {
	startTime     time.Time
	totalFiles    int
	workers       []*WorkerProgress
	mu            sync.RWMutex
	isRunning     bool
	stopChan      chan bool
	refreshRate   time.Duration
	termWidth     int               // Terminal width
	termHeight    int               // Terminal height
	statusLine    int               // Which line the status bar is on
	resizeChan    chan os.Signal    // Channel for terminal resize signals
	lastStatus    string            // Last rendered status to avoid unnecessary updates
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

type WorkerStatus int

const (
	StatusIdle WorkerStatus = iota
	StatusProcessing
	StatusError
	StatusCompleted
)

func (s WorkerStatus) String() string {
	switch s {
	case StatusIdle:
		return "IDLE"
	case StatusProcessing:
		return "PROC"
	case StatusError:
		return "ERR"
	case StatusCompleted:
		return "DONE"
	default:
		return "UNK"
	}
}

func NewProgressDisplay(workerCount, totalFiles int) *ProgressDisplay {
	workers := make([]*WorkerProgress, workerCount)
	for i := range workers {
		workers[i] = &WorkerProgress{
			ID:           i,
			Status:       StatusIdle,
			LastActivity: time.Now(),
		}
	}

	pd := &ProgressDisplay{
		startTime:   time.Now(),
		totalFiles:  totalFiles,
		workers:     workers,
		refreshRate: 1000 * time.Millisecond, // Reduced frequency to reduce flashing
		stopChan:    make(chan bool),
		resizeChan:  make(chan os.Signal, 1),
		lastStatus:  "",
	}
	
	// Get terminal size
	if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		pd.termWidth = width
		pd.termHeight = height
		pd.statusLine = height // Bottom line
	} else {
		// Fallback values if terminal size detection fails
		pd.termWidth = 80
		pd.termHeight = 24
		pd.statusLine = 24
	}
	
	return pd
}

func (pd *ProgressDisplay) Start() {
	pd.mu.Lock()
	pd.isRunning = true
	pd.mu.Unlock()

	// Bottom status mode - hide cursor and reserve bottom line
	fmt.Print("\033[?25l")
	pd.initBottomStatus()
	pd.startResizeMonitoring()

	go pd.displayLoop()
}

func (pd *ProgressDisplay) initBottomStatus() {
	// Move cursor to bottom line and clear it
	fmt.Printf("\033[%d;1H\033[2K", pd.statusLine)
	// Move cursor back up to allow logs to flow normally
	fmt.Printf("\033[%d;1H", pd.statusLine-1)
}

func (pd *ProgressDisplay) startResizeMonitoring() {
	if pd.resizeChan == nil {
		return
	}
	
	// Listen for terminal resize signals
	signal.Notify(pd.resizeChan, syscall.SIGWINCH)
	
	go func() {
		for {
			select {
			case <-pd.resizeChan:
				pd.handleResize()
			case <-pd.stopChan:
				return
			}
		}
	}()
}

func (pd *ProgressDisplay) handleResize() {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	
	// Get new terminal size
	if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		pd.termWidth = width
		pd.termHeight = height
		pd.statusLine = height // Update bottom line position
		pd.lastStatus = ""     // Force re-render after resize
		
		// Clear and reinitialize status bar at new position
		fmt.Printf("\033[%d;1H\033[K", pd.statusLine)
	}
}

func (pd *ProgressDisplay) Stop() {
	pd.mu.Lock()
	pd.isRunning = false
	pd.mu.Unlock()

	pd.stopChan <- true

	// Stop resize monitoring and clear bottom status line
	if pd.resizeChan != nil {
		signal.Stop(pd.resizeChan)
		close(pd.resizeChan)
	}
	fmt.Printf("\033[%d;1H\033[K\033[?25h", pd.statusLine)
}

func (pd *ProgressDisplay) UpdateWorker(workerID int, filesProcessed int, recordsProcessed int64, batchesProcessed int64, currentFile string, status WorkerStatus) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if workerID >= 0 && workerID < len(pd.workers) {
		worker := pd.workers[workerID]

		elapsed := time.Since(worker.LastActivity).Seconds()
		if elapsed > 0 {
			recordsDelta := recordsProcessed - worker.RecordsProcessed
			worker.ProcessingRate = float64(recordsDelta) / elapsed
		}

		worker.FilesProcessed = filesProcessed
		worker.RecordsProcessed = recordsProcessed
		worker.BatchesProcessed = batchesProcessed
		worker.CurrentFile = currentFile
		worker.Status = status
		worker.LastActivity = time.Now()
	}
}

func (pd *ProgressDisplay) UpdateWorkerWithTiming(workerID int, filesProcessed int, recordsProcessed int64, batchesProcessed int64, currentFile string, status WorkerStatus, avgTimePerFile time.Duration, totalFileTime time.Duration) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if workerID >= 0 && workerID < len(pd.workers) {
		worker := pd.workers[workerID]

		elapsed := time.Since(worker.LastActivity).Seconds()
		if elapsed > 0 {
			recordsDelta := recordsProcessed - worker.RecordsProcessed
			worker.ProcessingRate = float64(recordsDelta) / elapsed
		}

		worker.FilesProcessed = filesProcessed
		worker.RecordsProcessed = recordsProcessed
		worker.BatchesProcessed = batchesProcessed
		worker.CurrentFile = currentFile
		worker.Status = status
		worker.LastActivity = time.Now()
		worker.AvgTimePerFile = avgTimePerFile
		worker.TotalFileTime = totalFileTime
	}
}

func (pd *ProgressDisplay) displayLoop() {
	ticker := time.NewTicker(pd.refreshRate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pd.renderBottomStatus()
		case <-pd.stopChan:
			return
		}
	}
}

func (pd *ProgressDisplay) renderBottomStatus() {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	elapsed := time.Since(pd.startTime)
	totalFiles := pd.getTotalFilesProcessed()
	totalRecords := pd.getTotalRecordsProcessed()
	
	overallProgress := float64(totalFiles) / float64(pd.totalFiles) * 100
	if pd.totalFiles == 0 {
		overallProgress = 0
	}

	// Count active workers
	activeWorkers := 0
	for _, worker := range pd.workers {
		if worker.Status == StatusProcessing {
			activeWorkers++
		}
	}

	// Calculate processing rate
	var rateText string
	if elapsed.Seconds() > 0 {
		rate := float64(totalRecords) / elapsed.Seconds()
		rateText = fmt.Sprintf(" | %s/s", formatNumber(int64(rate)))
	}

	// Build compact progress line
	progressBar := pd.buildProgressBar(overallProgress, 20)
	
	progressLine := fmt.Sprintf("⚡ %s %.1f%% (%d/%d files) | %d workers | %s records | %v%s",
		progressBar,
		overallProgress,
		totalFiles,
		pd.totalFiles,
		activeWorkers,
		formatNumber(totalRecords),
		elapsed.Round(time.Second),
		rateText,
	)

	// Only update if the status has actually changed
	if progressLine == pd.lastStatus {
		return
	}
	pd.lastStatus = progressLine

	// Use a more stable approach without cursor save/restore
	fmt.Printf("\033[%d;1H\033[K%s\033[1G", pd.statusLine, progressLine)
}

func (pd *ProgressDisplay) buildProgressBar(percentage float64, width int) string {
	filled := int(percentage / 100 * float64(width))
	if filled > width {
		filled = width
	}

	accent := color.New(color.FgCyan).SprintFunc()
	muted := color.New(color.FgHiBlack).SprintFunc()

	var bar strings.Builder
	bar.WriteString("[")
	
	if filled > 0 {
		bar.WriteString(accent(strings.Repeat("█", filled)))
	}
	
	remaining := width - filled
	if remaining > 0 {
		bar.WriteString(muted(strings.Repeat("░", remaining)))
	}
	
	bar.WriteString("]")
	return bar.String()
}

func (pd *ProgressDisplay) getTotalFilesProcessed() int {
	total := 0
	for _, worker := range pd.workers {
		total += worker.FilesProcessed
	}
	return total
}

func (pd *ProgressDisplay) getTotalRecordsProcessed() int64 {
	var total int64
	for _, worker := range pd.workers {
		total += worker.RecordsProcessed
	}
	return total
}

func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	} else if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	} else if n < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	return fmt.Sprintf("%.1fB", float64(n)/1000000000)
}

func (pd *ProgressDisplay) UpdateFromPoolMetrics(metrics PoolMetrics, workerMetrics []WorkerMetrics) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	for i, wm := range workerMetrics {
		if i < len(pd.workers) {
			status := StatusProcessing
			if wm.ErrorCount > 0 {
				status = StatusError
			} else if wm.FilesProcessed == 0 {
				status = StatusIdle
			}

			pd.workers[i].FilesProcessed = wm.FilesProcessed
			pd.workers[i].RecordsProcessed = int64(wm.TotalRecords)
			pd.workers[i].BatchesProcessed = int64(wm.TotalBatches)
			pd.workers[i].Status = status
			pd.workers[i].LastActivity = wm.LastActivity
			pd.workers[i].AvgTimePerFile = wm.AvgTimePerFile
			pd.workers[i].TotalFileTime = wm.TotalFileTime
		}
	}
}