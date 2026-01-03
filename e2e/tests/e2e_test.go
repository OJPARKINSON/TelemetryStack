package tests

import (
	"context"
	"fmt"
	"log"
	"sort"
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
			name:            "1M_Records",
			numBatches:      100,
			recordsPerBatch: 10000,
			maxWaitTime:     2 * time.Minute,
			short:           true,
		},
		{
			name:            "5M_Records",
			numBatches:      500,
			recordsPerBatch: 10000,
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

			metrics, err := waitForRecordCountWithMetrics(expectedCount, 5*time.Minute)
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
			if metrics.AvgThroughput() < 50000 {
				t.Errorf("Throughput below target: %.0f < 50000 rec/sec",
					metrics.AvgThroughput())
			}
		})
	}

}

func waitForRecordCountWithMetrics(expectedCount int, timeout time.Duration) (*ThroughputMetrics, error) {
	metrics := &ThroughputMetrics{
		StartTime: time.Now(),
		Samples:   []Sample{},
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastCount := 0
	lastTime := time.Now()
	firstMeasurement := true // Add this flag

	for {
		now := time.Now()
		count, err := verification.GetRecordCount()
		if err != nil {
			return nil, err
		}

		if !firstMeasurement {
			deltaRecords := count - lastCount
			deltaTime := now.Sub(lastTime).Seconds()

			var instantThroughput float64
			if deltaTime > 0.1 {
				instantThroughput = float64(deltaRecords) / deltaTime
			}

			metrics.Samples = append(metrics.Samples, Sample{
				Timestamp:  now,
				Count:      count,
				Throughput: instantThroughput,
			})

			progress := float64(count) / float64(expectedCount) * 100
			log.Printf("Progress: %d/%d (%.1f%%) | Throughput: %.0f rec/sec",
				count, expectedCount, progress, instantThroughput)
		} else {
			firstMeasurement = false
			log.Printf("Starting monitoring - initial count: %d", count)
		}

		if count >= expectedCount {
			metrics.EndTime = now
			metrics.TotalRecords = count
			return metrics, nil
		}

		if now.After(deadline) {
			return metrics, fmt.Errorf("timeout: %d/%d records after %v",
				count, expectedCount, timeout)
		}

		lastCount = count
		lastTime = now

		<-ticker.C
	}
}

type ThroughputMetrics struct {
	StartTime    time.Time
	EndTime      time.Time
	TotalRecords int
	Samples      []Sample
}

type Sample struct {
	Timestamp  time.Time
	Count      int
	Throughput float64
}

func (m *ThroughputMetrics) AvgThroughput() float64 {
	if m.EndTime.IsZero() {
		return 0
	}
	return float64(m.TotalRecords) / m.EndTime.Sub(m.StartTime).Seconds()
}

func (m *ThroughputMetrics) PeakThroughput() float64 {
	peak := 0.0
	for _, s := range m.Samples {
		if s.Throughput > peak {
			peak = s.Throughput
		}
	}
	return peak
}

func (m *ThroughputMetrics) P95Throughput() float64 {
	if len(m.Samples) == 0 {
		return 0
	}

	throughputs := make([]float64, len(m.Samples))
	for i, s := range m.Samples {
		throughputs[i] = s.Throughput
	}
	sort.Float64s(throughputs)

	idx := int(float64(len(throughputs)) * 0.95)
	return throughputs[idx]
}
