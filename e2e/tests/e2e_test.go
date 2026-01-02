package tests

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/ojparkinson/IRacing-Display/e2e/pkg/containers"
	"github.com/ojparkinson/IRacing-Display/e2e/pkg/publisher"
	"github.com/ojparkinson/IRacing-Display/e2e/pkg/verification"
)

func TestAllTicksAreStored(t *testing.T) {
	testCases := []struct {
		name            string
		numBatches      int
		recordsPerBatch int
		maxWaitTime     time.Duration
	}{
		{
			name:            "1M_Records",
			numBatches:      100,
			recordsPerBatch: 10000,
			maxWaitTime:     2 * time.Minute,
		},
		{
			name:            "5M_Records",
			numBatches:      500,
			recordsPerBatch: 10000,
			maxWaitTime:     10 * time.Minute,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			network, _ := containers.CreateNetwork(ctx)

			containers.SpinUpQuestDB(t, ctx, network)
			rabbitmqC := containers.StartRabbitMQ(t, ctx, network)
			containers.StartTelemetryService(t, ctx, network)

			batches := publisher.GenerateBatch(tc.numBatches, tc.recordsPerBatch)

			pub, err := publisher.NewPublisher(rabbitmqC, ctx)
			if err != nil {
				t.Fatalf("Failed to create publisher: %v", err)
			}

			startTime := time.Now()
			pub.PublishBatch(rabbitmqC, batches, ctx)
			publishDuration := time.Since(startTime)

			expectedCount := tc.numBatches * tc.recordsPerBatch
			if err := waitForRecordCount(ctx, expectedCount, tc.maxWaitTime); err != nil {
				t.Fatalf("Failed to verify record count: %v", err)
			}

			totalDuration := time.Since(startTime)
			throughput := float64(expectedCount) / totalDuration.Seconds()

			t.Logf("✓ Published %d records in %v (%.0f records/sec)",
				expectedCount, publishDuration, throughput)
			t.Logf("✓ E2E processing completed in %v", totalDuration)
		})
	}

}

func waitForRecordCount(ctx context.Context, expectedCount int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		count, err := verification.GetRecordCount()
		if err != nil {
			return fmt.Errorf("failed to query count: %w", err)
		}

		if count == expectedCount {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout: expected %d records, got %d after %v",
				expectedCount, count, timeout)
		}

		log.Printf("Progress: %d/%d records (%.1f%%)",
			count, expectedCount, float64(count)/float64(expectedCount)*100)

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
