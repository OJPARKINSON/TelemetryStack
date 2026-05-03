package tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ojparkinson/IRacing-Display/e2e/pkg/containers"
	mockPublisher "github.com/ojparkinson/IRacing-Display/e2e/pkg/publisher"
	"github.com/ojparkinson/IRacing-Display/e2e/pkg/verification"
	"github.com/testcontainers/testcontainers-go"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

func TestAllTicksAreStored(t *testing.T) {
	testCases := []struct {
		name            string
		numBatches      int
		recordsPerBatch int
		maxWaitTime     time.Duration
		short           bool
	}{
		{
			name:            "2.5M_Records",
			numBatches:      100,
			recordsPerBatch: 25000,
			maxWaitTime:     2 * time.Minute,
			short:           false,
		},
		{
			name:            "5M_Records",
			numBatches:      200,
			recordsPerBatch: 25000,
			maxWaitTime:     10 * time.Minute,
			short:           true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if testing.Short() && tc.short {
				t.Skip()
			}

			ctx := context.Background()
			network, _ := containers.CreateNetwork(ctx)
			testcontainers.CleanupNetwork(t, network)

			containers.SpinUpQuestDB(t, ctx, network)

			telemetryService := containers.StartTelemetryService(t, ctx, network)

			batches := mockPublisher.GenerateBatch(tc.numBatches, tc.recordsPerBatch)

			pub, err := mockPublisher.NewPublisher(ctx, telemetryService)
			if err != nil {
				t.Fatalf("Failed to create publisher: %v", err)
			}

			publishStart := time.Now()
			pub.PublishBatch(ctx, batches)

			publishDuration := time.Since(publishStart)
			expectedCount := tc.numBatches * tc.recordsPerBatch
			publishThroughput := float64(expectedCount) / publishDuration.Seconds()

			metrics, err := verification.WaitForRecordCountWithMetrics(expectedCount, 5*time.Minute)
			if err != nil {
				t.Fatalf("Processing failed: %v", err)
			}

			// Report
			p := message.NewPrinter(language.English)

			publishThroughputDisplay := strings.ReplaceAll(p.Sprintf("%v", number.Decimal(publishThroughput)), ",", "_")
			avgThroughput := strings.ReplaceAll(p.Sprintf("%v", number.Decimal(metrics.AvgThroughput())), ",", "_")
			peakThroughput := strings.ReplaceAll(p.Sprintf("%v", number.Decimal(metrics.PeakThroughput())), ",", "_")
			p95Throughput := strings.ReplaceAll(p.Sprintf("%v", number.Decimal(metrics.P95Throughput())), ",", "_")

			t.Logf("📊 Throughput Metrics:")
			t.Logf("  Publisher:  %s rec/sec", publishThroughputDisplay)
			t.Logf("  E2E Avg:    %s rec/sec", avgThroughput)
			t.Logf("  E2E Peak:   %s rec/sec", peakThroughput)
			t.Logf("  E2E P95:    %s rec/sec", p95Throughput)

			if metrics.P95Throughput() < 25000 {
				t.Errorf("P95 throughput below target: %.0f < 25000 rec/sec",
					metrics.P95Throughput())
			}

			err2 := verification.TunicateTable()
			if err2 != nil {
				log.Fatal("error TunicateTable, ", err)
			}
		})
	}
}
func TestFixedFilesProcessedSpeed(t *testing.T) {
	ctx := context.Background()
	network, _ := containers.CreateNetwork(ctx)
	testcontainers.CleanupNetwork(t, network)

	// Discover .ibt files
	ibtPath, err := filepath.Abs("../../ingest/go/ibt_files")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	dirEntries, err := os.ReadDir(ibtPath)
	if err != nil {
		t.Fatalf("Failed to read ibt directory: %v", err)
	}

	// Count .ibt files
	ibtFileCount := 0
	for _, entry := range dirEntries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".ibt" {
			ibtFileCount++
		}
	}

	if ibtFileCount == 0 {
		t.Fatal("No .ibt files found to process")
	}

	t.Logf("Found %d .ibt files to process", ibtFileCount)

	// Start containers
	containers.SpinUpQuestDB(t, ctx, network)
	containers.StartTelemetryService(t, ctx, network)

	// Get initial count from QuestDB
	initialCount, err := verification.GetRecordCount()
	if err != nil {
		t.Fatalf("Failed to get initial record count: %v", err)
	}
	t.Logf("Initial DB record count: %d", initialCount)

	// Record start time
	ingestStart := time.Now()

	// Run the ingest app
	t.Logf("🚀 Starting ingest process...")
	ingestCmd := fmt.Sprintf("cd ../../ingest/go && go run cmd/ingest/main.go -p=\"%s\"", ibtPath)

	// Run ingest in background and capture output
	ingestResult := make(chan error, 1)
	go func() {
		// Use bash -c to run the cd && go run command
		cmd := fmt.Sprintf("bash -c '%s'", ingestCmd)
		err := runCommand(cmd, 10*time.Minute)
		ingestResult <- err
	}()

	// Poll QuestDB to detect when ingestion completes
	t.Logf("📊 Monitoring QuestDB for record count changes...")
	var finalCount int
	stableCount := 0
	lastCount := initialCount
	pollInterval := 2 * time.Second
	maxStablePolls := 5 // Consider complete if count stable for 5 polls (10 seconds)

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()

	ingestComplete := false
	for !ingestComplete {
		select {
		case err := <-ingestResult:
			if err != nil {
				t.Fatalf("Ingest process failed: %v", err)
			}
			t.Logf("✅ Ingest process completed")
			// Continue polling until count stabilizes
		case <-pollTicker.C:
			currentCount, err := verification.GetRecordCount()
			if err != nil {
				t.Logf("Warning: failed to get record count: %v", err)
				continue
			}

			if currentCount > lastCount {
				t.Logf("  Records in DB: %d (+%d)", currentCount, currentCount-lastCount)
				lastCount = currentCount
				stableCount = 0
			} else if currentCount == lastCount {
				stableCount++
				if stableCount >= maxStablePolls {
					finalCount = currentCount
					ingestComplete = true
					t.Logf("  Record count stable at %d", finalCount)
				}
			}
		case <-time.After(15 * time.Minute):
			t.Fatalf("Timeout waiting for ingestion to complete")
		}
	}

	// Calculate end-to-end time
	e2eDuration := time.Since(ingestStart)
	totalRecords := finalCount - initialCount

	t.Logf("")
	t.Logf("📊 End-to-End Performance Metrics:")
	t.Logf("  Total records processed: %d", totalRecords)
	t.Logf("  Total E2E time:          %v", e2eDuration)
	t.Logf("  Average throughput:      %.0f rec/sec", float64(totalRecords)/e2eDuration.Seconds())

	// Cleanup
	err2 := verification.TunicateTable()
	if err2 != nil {
		log.Fatal("error TunicateTable, ", err)
	}
}

// runCommand runs a shell command with a timeout
func runCommand(cmdString string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Extract the actual command from the bash -c wrapper
	actualCmd := cmdString[len("bash -c '") : len(cmdString)-1]

	log.Printf("Executing: %s", actualCmd)

	// Create the command with bash -c
	cmd := exec.CommandContext(ctx, "bash", "-c", actualCmd)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Command failed: %v\nOutput: %s", err, string(output))
		return err
	}

	log.Printf("Command completed. Output:\n%s", string(output))
	return nil
}
