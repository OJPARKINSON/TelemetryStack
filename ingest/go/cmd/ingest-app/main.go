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
		oldProcs := runtime.GOMAXPROCS(cfg.GoMaxProcs)
		logger.Info("GOMAXPROCS configured",
			zap.Int("old", oldProcs),
			zap.Int("new", cfg.GoMaxProcs))
	} else {
		logger.Info("Using Go default GOMAXPROCS",
			zap.Int("cpus", runtime.GOMAXPROCS(0)))
	}

	// Enable profiling if ENABLE_PPROF environment variable is set
	if os.Getenv("ENABLE_PPROF") == "true" {
		go func() {
			logger.Info("Starting pprof server on :6060")
			if err := http.ListenAndServe(":6060", nil); err != nil {
				logger.Error("pprof server failed", zap.Error(err))
			}
		}()
	}

	// Enable CPU profiling if CPU_PROFILE environment variable is set
	if cpuProfile := os.Getenv("CPU_PROFILE"); cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			logger.Fatal("Could not create CPU profile", zap.Error(err))
		}
		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			logger.Fatal("Could not start CPU profile", zap.Error(err))
		}
		defer pprof.StopCPUProfile()
		logger.Info("CPU profiling enabled", zap.String("file", cpuProfile))
	}

	// Setup context and signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalCh
		logger.Info("Graceful shutdown initiated - allowing data to be flushed...")
		cancel()
	}()

	// Get telemetry folder from remaining args after flag parsing
	args := flag.Args()
	if len(args) < 1 {
		logger.Fatal("Usage: ./ingest-app [options] <telemetry-folder-path>")
	}

	telemetryFolder := args[0]
	if !strings.HasSuffix(telemetryFolder, string(filepath.Separator)) {
		telemetryFolder += string(filepath.Separator)
	}

	// Create worker pool
	pool := worker.NewWorkerPool(cfg, logger)

	// Discover and queue files
	expectedFiles, err := discoverAndQueueFiles(ctx, pool, telemetryFolder, cfg, logger)
	if err != nil {
		logger.Error("Error during file discovery", zap.Error(err))
		return
	}

	// Initialize progress display
	if !*quiet {
		progress = worker.NewProgressDisplay(cfg.WorkerCount, expectedFiles)
		pool.SetProgressDisplay(progress)
		progress.Start()
		defer progress.Stop()

		logger.Info("Starting ingest service",
			zap.Int("workers", cfg.WorkerCount),
			zap.Int("files", expectedFiles))
	} else {
		logger.Info("Starting ingest service in quiet mode",
			zap.Int("workers", cfg.WorkerCount),
			zap.Int("files", expectedFiles))
	}

	// Start worker pool
	pool.Start()
	defer pool.Stop()

	// Wait for completion
	waitForCompletion(ctx, pool, startTime, expectedFiles, *quiet)

	if !*quiet {
		logger.Info("Processing completed", zap.Duration("duration", time.Since(startTime)))
	}

	// Write memory profile if MEM_PROFILE environment variable is set
	if memProfile := os.Getenv("MEM_PROFILE"); memProfile != "" {
		f, err := os.Create(memProfile)
		if err != nil {
			logger.Error("Could not create memory profile", zap.Error(err))
		} else {
			defer f.Close()
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				logger.Error("Could not write memory profile", zap.Error(err))
			} else {
				logger.Info("Memory profile written", zap.String("file", memProfile))
			}
		}
	}

	// Give a moment for final logs to flush
	time.Sleep(2 * time.Second)
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
			if !quiet {
				logger.Info("Shutdown requested")
			}
			return
		default:
			time.Sleep(20 * time.Millisecond)
			metrics := pool.GetMetrics()

			if metrics.QueueDepth == 0 && metrics.TotalFilesProcessed >= expectedFiles {
				if !quiet {
					logger.Info("All files processed",
						zap.Int("total_files", expectedFiles),
						zap.Duration("duration", time.Since(startTime)))
				}
				return
			}
		}
	}
}
