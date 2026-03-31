package mockPublisher

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime"
	sync "sync"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"google.golang.org/protobuf/proto"
)

type Publisher struct {
	serverUrl string
}

func NewPublisher(ctx context.Context, telemetryService *testcontainers.DockerContainer) (*Publisher, error) {
	host, _ := telemetryService.Host(ctx)
	port, _ := telemetryService.MappedPort(ctx, "8010")

	serverUrl := fmt.Sprintf("%s:%s", host, port.Port())

	return &Publisher{
		serverUrl: serverUrl,
	}, nil
}

func (p *Publisher) PublishBatch(ctx context.Context, batches []*TelemetryBatch) {
	numWorkers := runtime.NumCPU() // Use all CPUs
	if numWorkers > len(batches)/10 {
		numWorkers = len(batches) / 10
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	workChan := make(chan *TelemetryBatch, len(batches))
	errChan := make(chan error, len(batches))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		client := http.Client{Timeout: 10 * time.Second}

		go func(workerID int) {
			defer wg.Done()
			published := 0

			for batch := range workChan {
				data, err := proto.Marshal(batch)
				if err != nil {
					errChan <- fmt.Errorf("worker %d: marshal error: %w", workerID, err)
					continue
				}

				dataReader := bytes.NewReader(data)

				res, err := client.Post("http://"+p.serverUrl+"/api/ingest", "application/x-protobuf", dataReader)
				if err != nil {
					log.Printf("Worker %d: Failed to publish batch %s: %v", batch.WorkerId, batch.BatchId, err)

					continue
				}
				defer res.Body.Close()

				if res.StatusCode >= 200 && res.StatusCode < 300 {
					published++
				}
			}

			fmt.Printf("Worker %d published %d batches\n", workerID, published)
		}(i)
	}

	// Send all batches to workers
	start := time.Now()
	for _, batch := range batches {
		workChan <- batch
	}
	close(workChan)

	// Wait for completion
	wg.Wait()
	close(errChan)

	// Report errors
	errorCount := 0
	for err := range errChan {
		fmt.Println(err)
		errorCount++
	}

	elapsed := time.Since(start)
	fmt.Printf("✅ Published %d batches in %v (%.0f batches/sec, %d errors)\n",
		len(batches), elapsed, float64(len(batches))/elapsed.Seconds(), errorCount)
}
