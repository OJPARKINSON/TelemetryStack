/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/processing"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/worker"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	fresh    bool
	progress *worker.ProgressDisplay
	logger   *zap.Logger
)

var processCmd = &cobra.Command{
	Use:   "go",
	Short: "Process the telemetry data in the background",
	Long: `Watch the telemetry directory and in the background process new telemetry files
	
	To clean the cache of sent file run with --fresh to upload all data in the dir again`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Inside rootCmd Run with args: %v\n", args)
		Process(args[0])
	},
}

func init() {
	processCmd.Flags().BoolVarP(&display, "display", "d", true, "terminal display of the ingest process")
	processCmd.Flags().StringVarP(&telemetryPath, "telemetryPath", "p", "", "path to IRacing telemetry folder")

	processCmd.Flags().BoolVarP(&fresh, "fresh", "f", false, "will clean the local store of files that have been processed and start from fresh")
}

func Process(telemetryFolder string) {
	var quiet = flag.Bool("quiet", false, "Disable progress display")
	var verbose = flag.Bool("verbose", false, "Enable verbose logging")

	startTime := time.Now()

	// Initialize Zap logger
	var err error
	if *verbose {
		// Verbose mode: full development logging
		logger, err = zap.NewDevelopment()
	} else {
		// Silent mode: errors and above
		config := zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
		logger, err = config.Build()
	}
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration
	cfg := config.LoadConfig()

	// Apply GOMAXPROCS if explicitly configured (0 means use Go's default)
	if cfg.GoMaxProcs > 0 {
		runtime.GOMAXPROCS(cfg.GoMaxProcs)
	}

	if os.Getenv("ENABLE_PPROF") == "true" {
		go func() {
			if err := http.ListenAndServe(":6060", nil); err != nil {
				logger.Error("pprof server failed",
					zap.Error(err),
					zap.String("action", "Check port 6060 is not in use"))
			}
		}()
	}

	if cpuProfile := os.Getenv("CPU_PROFILE"); cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			logger.Fatal("Could not create CPU profile",
				zap.Error(err),
				zap.String("path", cpuProfile),
				zap.String("action", "Check directory exists and has write permissions"))
		}
		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			logger.Fatal("Could not start CPU profile",
				zap.Error(err),
				zap.String("action", "Check file can be written"))
		}
		defer pprof.StopCPUProfile()
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
			zap.String("action", "Create directory or set IBT_DATA_DIR environment variable"))
	}

	// Create worker pool
	pool := worker.NewWorkerPool(cfg, logger)

	expectedFiles, err := discoverAndQueueFiles(ctx, pool, telemetryFolder, cfg, logger)
	if err != nil {
		logger.Error("File discovery failed",
			zap.Error(err),
			zap.String("path", telemetryFolder),
			zap.String("action", "Check directory permissions and IBT files exist"))
		return
	}
	log.Printf("STARTUP: Found %d IBT files to process", expectedFiles)

	// Initialize progress display
	if !*quiet {
		progress = worker.NewProgressDisplay(cfg.WorkerCount, expectedFiles)
		pool.SetProgressDisplay(progress)
		progress.Start()
		defer progress.Stop()
	}

	// Start worker pool
	if err := pool.Start(); err != nil {
		logger.Fatal("Failed to start worker pool",
			zap.Error(err),
			zap.String("action", "Check system resources and configuration"))
	}
	defer func() {
		if err := pool.Stop(); err != nil {
			logger.Error("Error stopping worker pool",
				zap.Error(err))
		}
	}()

	// Wait for completion
	waitForCompletion(ctx, pool, startTime, expectedFiles, *quiet)

	// Write memory profile if MEM_PROFILE environment variable is set
	if memProfile := os.Getenv("MEM_PROFILE"); memProfile != "" {
		f, err := os.Create(memProfile)
		if err != nil {
			logger.Error("Could not create memory profile",
				zap.Error(err),
				zap.String("path", memProfile),
				zap.String("action", "Check directory exists and has write permissions"))
		} else {
			defer f.Close()
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				logger.Error("Could not write memory profile",
					zap.Error(err),
					zap.String("action", "Check disk space and file permissions"))
			}
		}
	}
}

func discoverAndQueueFiles(ctx context.Context, pool *worker.WorkerPool, telemetryFolder string, cfg *config.Config, logger *zap.Logger) (int, error) {
	directory := processing.NewDir(telemetryFolder, cfg, logger)
	files := directory.WatchDir()

	filesQueued := 0
	for _, file := range files {
		select {
		case <-ctx.Done():
			return filesQueued, ctx.Err()
		default:
		}

		fileName := file.Name()

		if !strings.Contains(fileName, ".ibt") {
			continue
		}

		workItem := worker.WorkItem{
			FilePath:   filepath.Join(telemetryFolder, fileName),
			FileInfo:   file,
			RetryCount: 0,
		}

		if err := pool.SubmitFile(workItem); err != nil {
			return filesQueued, err
		}

		filesQueued++
	}

	return filesQueued, nil
}

func waitForCompletion(ctx context.Context, pool *worker.WorkerPool, startTime time.Time, expectedFiles int, quiet bool) {
	for {
		select {
		case <-ctx.Done():
			// Shutdown requested - no log needed, handled by pool
			return
		default:
			time.Sleep(20 * time.Millisecond)
			metrics := pool.GetMetrics()

			if metrics.QueueDepth == 0 && metrics.TotalFilesProcessed >= expectedFiles {
				// Completion - metrics available via Prometheus, no log needed
				return
			}
		}
	}
}
