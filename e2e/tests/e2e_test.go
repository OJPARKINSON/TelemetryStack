package tests

import (
	"context"
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
		short           bool
	}{
		{
			name:            "500k_Records",
			numBatches:      20,
			recordsPerBatch: 25000,
			maxWaitTime:     1 * time.Minute,
			short:           true,
		},
		{
			name:            "1M_Records",
			numBatches:      40,
			recordsPerBatch: 25000,
			maxWaitTime:     2 * time.Minute,
			short:           false, // swap back to true
		},
		{
			name:            "5M_Records",
			numBatches:      200,
			recordsPerBatch: 25000,
			maxWaitTime:     10 * time.Minute,
			short:           false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if testing.Short() && tc.short {
				t.Skip()
			}

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

			publishStart := time.Now()
			pub.PublishBatch(rabbitmqC, batches, ctx)

			publishDuration := time.Since(publishStart)
			expectedCount := tc.numBatches * tc.recordsPerBatch
			publishThroughput := float64(expectedCount) / publishDuration.Seconds()

			metrics, err := verification.WaitForRecordCountWithMetrics(expectedCount, 5*time.Minute)
			if err != nil {
				t.Fatalf("Processing failed: %v", err)
			}

			// Report
			t.Logf("📊 Throughput Metrics:")
			t.Logf("  Publisher:  %.0f rec/sec", publishThroughput)
			t.Logf("  E2E Avg:    %.0f rec/sec", metrics.AvgThroughput())
			t.Logf("  E2E Peak:   %.0f rec/sec", metrics.PeakThroughput())
			t.Logf("  E2E P95:    %.0f rec/sec", metrics.P95Throughput())

			// Assert minimum performance
			if metrics.P95Throughput() < 50000 {
				t.Errorf("Throughput below target: %.0f < 50000 rec/sec",
					metrics.P95Throughput())
			}

			err2 := verification.TunicateTable()
			if err2 != nil {
				log.Fatal("error TunicateTable, ", err)
			}
		})
	}

}
