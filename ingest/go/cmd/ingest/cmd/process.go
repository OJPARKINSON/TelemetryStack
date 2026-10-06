/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/metrics"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/processing"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/ui"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/worker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"golang.org/x/term"
)

var logger *zap.Logger

func Process(telemetryFolder string) error {
	envErr := godotenv.Load()
	if envErr != nil {
		log.Fatal("Error loading .env file")
	}

	startTime := time.Now()

	// Full logs only in verbose mode; otherwise just errors.
	logCfg := zap.NewDevelopmentConfig()
	if !verbose {
		logCfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
		logCfg.DisableStacktrace = true
		log.SetOutput(io.Discard)
	}
	var err error
	logger, err = logCfg.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
		return err
	}

	defer logger.Sync()

	// Load configuration
	cfg := config.LoadConfig()

	// Apply GOMAXPROCS if explicitly configured (0 means use Go's default)
	if cfg.GoMaxProcs > 0 {
		runtime.GOMAXPROCS(cfg.GoMaxProcs)
	}

	// Start metrics pusher
	if !cfg.DryRun {
		metrics.StartPusher(cfg.ServerUrl + "")
	}

	// Setup context and signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalCh
		cancel()
	}()

	if !strings.HasSuffix(telemetryFolder, string(filepath.Separator)) {
		telemetryFolder += string(filepath.Separator)
	}

	// Verify telemetry folder exists
	if _, err := os.Stat(telemetryFolder); os.IsNotExist(err) {
		logger.Fatal("Telemetry directory does not exist",
			zap.String("path", telemetryFolder),
			zap.String("action", "Create directory or set path attribute"))
		return err
	}

	items := discoverFiles(telemetryFolder, cfg, logger)
	files := uiFiles(items)
	log.Printf("STARTUP: Found %d IBT files to process", len(items))

	// Progress view: TUI on a terminal, plain lines when piped, nothing in verbose mode.
	var reporter *ui.Reporter
	poolLogger := logger
	uiDone := make(chan struct{})
	switch {
	case verbose:
		close(uiDone)
	case term.IsTerminal(int(os.Stdout.Fd())):
		program := tea.NewProgram(ui.NewModel(files, cancel))
		reporter = ui.NewTUIReporter(program)
		poolLogger = zap.NewNop() // nothing may log while the TUI owns the terminal; failed rows show in it instead
		go func() {
			defer close(uiDone)
			if _, err := program.Run(); err != nil {
				fmt.Fprintln(os.Stderr, "progress view:", err)
			}
		}()
	default:
		reporter = ui.NewPlainReporter(os.Stdout, files)
		close(uiDone)
	}

	var progress processing.ProgressCallback
	if reporter != nil {
		progress = reporter
	}
	pool := worker.NewWorkerPool(cfg, poolLogger, progress)

	// Start workers first so a backlog larger than the queue drains instead of blocking the submit loop.
	if err := pool.Start(); err != nil {
		logger.Fatal("Failed to start worker pool",
			zap.Error(err),
			zap.String("action", "Check system resources and configuration"))
		return err
	}

	defer func() {
		if err := pool.Stop(); err != nil {
			logger.Error("Error stopping worker pool",
				zap.Error(err))
		}
	}()

	expectedFiles, err := queueFiles(ctx, pool, items)
	if err != nil {
		if reporter != nil {
			reporter.Done(time.Since(startTime), true)
		}
		<-uiDone
		if ctx.Err() != nil {
			return nil // interrupted while queueing
		}
		logger.Error("Queueing files failed",
			zap.Error(err),
			zap.String("path", telemetryFolder))
		return err
	}

	// Wait for completion
	waitForCompletion(ctx, pool, startTime, expectedFiles)
	if reporter != nil {
		reporter.Done(time.Since(startTime), ctx.Err() != nil)
	}
	<-uiDone

	// Write memory profile if MEM_PROFILE environment variable is set
	if memProfile := os.Getenv("MEM_PROFILE"); memProfile != "" {
		f, err := os.Create(memProfile)
		if err != nil {
			logger.Error("Could not create memory profile",
				zap.Error(err),
				zap.String("path", memProfile),
				zap.String("action", "Check directory exists and has write permissions"))
			return err
		} else {
			defer f.Close()
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				logger.Error("Could not write memory profile",
					zap.Error(err),
					zap.String("action", "Check disk space and file permissions"))
				return err
			}
		}
	}

	return nil
}

var timestampRegEx = regexp.MustCompile(`(\d{4}-\d{2}-\d{2}) (\d{2}-\d{2}-\d{2})`)

func parseTimestamp(filename string) (time.Time, error) {
	matching := timestampRegEx.FindStringSubmatch(filename)
	if matching == nil {
		return time.Time{}, fmt.Errorf("no timestamp found in %q", filename)
	}
	raw := matching[1] + "T" + matching[2] + "Z"
	return time.Parse("2006-01-02T15-04-05Z", raw)
}

// discoverFiles returns the .ibt files ready to upload as work items, oldest first.
func discoverFiles(telemetryFolder string, cfg *config.Config, logger *zap.Logger) []worker.WorkItem {
	entries := processing.NewDir(telemetryFolder, cfg, logger).WatchDir()

	items := make([]worker.WorkItem, 0, len(entries))
	for _, entry := range entries {
		if !strings.Contains(entry.Name(), ".ibt") {
			continue
		}
		at, err := parseTimestamp(entry.Name())
		if err != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, worker.WorkItem{
			FilePath: filepath.Join(telemetryFolder, entry.Name()),
			FileInfo: fs.FileInfoToDirEntry(info),
			FileDate: at,
		})
	}
	sort.SliceStable(items, func(a, b int) bool { return items[a].FileDate.Before(items[b].FileDate) })
	return items
}

func uiFiles(items []worker.WorkItem) []ui.File {
	files := make([]ui.File, len(items))
	for i, item := range items {
		info, _ := item.FileInfo.Info()
		files[i] = ui.File{Name: filepath.Base(item.FilePath), Size: info.Size()}
	}
	return files
}

// queueFiles submits items to the pool in order and returns how many were queued; it stops early when ctx is cancelled.
func queueFiles(ctx context.Context, pool *worker.WorkerPool, items []worker.WorkItem) (int, error) {
	for i, item := range items {
		if err := pool.SubmitFile(ctx, item); err != nil {
			return i, err
		}
	}
	return len(items), nil
}

func waitForCompletion(ctx context.Context, pool *worker.WorkerPool, startTime time.Time, expectedFiles int) {
	for {
		select {
		case <-ctx.Done():
			// Shutdown requested - no log needed, handled by pool
			return
		default:
			time.Sleep(20 * time.Millisecond)
			metrics := pool.GetMetrics()

			if metrics.QueueDepth == 0 && metrics.TotalFilesProcessed+metrics.TotalFilesFailed >= expectedFiles {
				// Completion - metrics available via Prometheus, no log needed
				return
			}
		}
	}
}
