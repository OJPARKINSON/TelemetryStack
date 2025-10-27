package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
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
	"go.uber.org/zap"
)

var progress *worker.ProgressDisplay
var logger *zap.Logger

func main() {
	// Define CLI flags
	var quiet = flag.Bool("quiet", false, "Disable progress display")
	var verbose = flag.Bool("verbose", false, "Enable verbose logging")
	var help = flag.Bool("help", false, "Show help")

	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	startTime := time.Now()

	// Initialize Zap logger
	var err error
	if *verbose {
		// Verbose mode: full development logging
		logger, err = zap.NewDevelopment()
	} else {
		// Silent mode: only fatal errors
		config := zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.FatalLevel)
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

	// Enable profiling if ENABLE_PPROF environment variable is set
	if os.Getenv("ENABLE_PPROF") == "true" {
		go func() {
			if err := http.ListenAndServe(":6060", nil); err != nil {
				logger.Error("pprof server failed",
					zap.Error(err),
					zap.String("action", "Check port 6060 is not in use"))
			}
		}()
	}

	// Enable CPU profiling if CPU_PROFILE environment variable is set
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

	// Get telemetry folder from args or use configured default
	var telemetryFolder string
	args := flag.Args()
	if len(args) >= 1 {
		telemetryFolder = args[0]
	} else {
		telemetryFolder = cfg.DataDirectory
	}

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

	// Discover and queue files
	expectedFiles, err := discoverAndQueueFiles(ctx, pool, telemetryFolder, cfg, logger)
	if err != nil {
		logger.Error("File discovery failed",
			zap.Error(err),
			zap.String("path", telemetryFolder),
			zap.String("action", "Check directory permissions and IBT files exist"))
		return
	}

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

func printHelp() {
	fmt.Println("IRacing Telemetry Ingest Service")
	fmt.Println()
	fmt.Println("Usage: ./ingest-app [options] <telemetry-folder-path>")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --quiet          Disable progress display")
	fmt.Println("  --verbose        Enable verbose logging (default: silent except errors)")
	fmt.Println("  --help           Show this help message")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  IBT_DATA_DIR              Data directory path (default: ./ibt_files/)")
	fmt.Println("  WORKER_COUNT              Number of parallel workers (default: CPU count + 25%)")
	fmt.Println("  BATCH_SIZE_BYTES          Batch size in bytes (default: 32MB)")
	fmt.Println("  BATCH_SIZE_RECORDS        Records per batch (default: 16000)")
	fmt.Println("  DISABLE_RABBITMQ          Set to 'true' to disable RabbitMQ (default: false)")
	fmt.Println("  RABBITMQ_URL              RabbitMQ connection URL")
	fmt.Println("  ENABLE_PPROF              Enable pprof profiling server on :6060")
	fmt.Println("  CPU_PROFILE               Write CPU profile to file")
	fmt.Println("  MEM_PROFILE               Write memory profile to file")
	fmt.Println("  GOMAXPROCS                Set GOMAXPROCS (default: auto)")
	fmt.Println()
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
