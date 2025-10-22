package dashboard

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
)

type dashboard struct {
	pw      progress.Writer
	metrics map[int]*WorkerMetrics
	mu      sync.RWMutex
	done    chan struct{}
}

type WorkerMetrics struct {
	// Worker identification
	ID       int
	WorkerID int

	// File processing stats
	FilesProcessed int
	CurrentFile    string

	// Record processing stats
	TotalRecords  int
	RecordsPerSec int

	// Batch stats
	TotalBatches int
	BatchesSent  int

	// Performance metrics
	Throughput        float64 // MB/s
	ThroughputMBps    float64 // MB/s (alternative name)
	AvgProcessingTime time.Duration

	// RabbitMQ queue stats
	QueueSize     int
	QueueCapacity int
	QueueUsage    int // Percentage 0-100

	// Status
	Status           string // "Processing", "Idle", "Error", etc.
	CircuitBreaker   bool
	FailedBatches    int
	PersistedBatches int

	// Timing
	StartTime  time.Time
	LastUpdate time.Time
}

func NewDashboard(workerCount int) *dashboard {
	pw := progress.NewWriter()
	pw.SetAutoStop(false)
	pw.SetTrackerLength(20)
	pw.SetUpdateFrequency(300 * time.Millisecond)
	pw.Style().Visibility.TrackerOverall = true

	return &dashboard{
		pw:      pw,
		metrics: make(map[int]*WorkerMetrics),
		done:    make(chan struct{}),
	}
}

func (d *dashboard) Start() {
	go d.pw.Render()
	go d.renderStats()
}

func (d *dashboard) renderStats() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.printTable()
		case <-d.done:
			return
		}
	}
}

func (d *dashboard) printTable() {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Clear screen and move cursor to top
	fmt.Print("\033[2J\033[H")

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Worker", "Files", "Records/s", "MB/s", "RabbitMQ Queue"})

	totalRecords := 0
	totalThroughput := 0.0

	for id, m := range d.metrics {
		queuePct := 0.0
		if m.QueueCapacity > 0 {
			queuePct = float64(m.QueueSize) / float64(m.QueueCapacity) * 100
		}

		t.AppendRow(table.Row{
			fmt.Sprintf("Worker %d", id),
			m.FilesProcessed,
			m.RecordsPerSec,
			fmt.Sprintf("%.2f", m.ThroughputMBps),
			fmt.Sprintf("%d/%d (%.0f%%)", m.QueueSize, m.QueueCapacity, queuePct),
		})

		totalRecords += m.RecordsPerSec
		totalThroughput += m.ThroughputMBps
	}

	t.AppendSeparator()
	t.AppendFooter(table.Row{"Total", "-", totalRecords, fmt.Sprintf("%.2f", totalThroughput), "-"})

	t.SetStyle(table.StyleColoredBright)
	t.Render()
}

func (d *dashboard) UpdateWorker(id int, metrics *WorkerMetrics) {
	d.mu.Lock()
	d.metrics[id] = metrics
	d.mu.Unlock()
}

func (d *dashboard) Stop() {
	close(d.done)
	d.pw.Stop()
}
