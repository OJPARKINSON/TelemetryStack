package publisher

import (
	"context"
	"fmt"
	"runtime"
	sync "sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go"
	"google.golang.org/protobuf/proto"
)

type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewPublisher(rabbitmq *testcontainers.DockerContainer, ctx context.Context) (*Publisher, error) {
	host, _ := rabbitmq.Host(ctx)
	port, _ := rabbitmq.MappedPort(ctx, "5672")

	conn, err := amqp.Dial(fmt.Sprintf("amqp://admin:changeme@%s:%s", host, port.Port()))
	if err != nil {
		return nil, err
	}

	fmt.Println("Connecting channel")
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &Publisher{
		conn:    conn,
		channel: channel,
	}, nil
}

func (p *Publisher) PublishBatch(rabbitmq *testcontainers.DockerContainer, batches []*TelemetryBatch, ctx context.Context) {
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
		go func(workerID int) {
			defer wg.Done()
			published := 0

			for batch := range workChan {
				data, err := proto.Marshal(batch)
				if err != nil {
					errChan <- fmt.Errorf("worker %d: marshal error: %w", workerID, err)
					continue
				}

				err = p.channel.PublishWithContext(ctx, "telemetry_topic", "telemetry.ticks",
					false, false,
					amqp.Publishing{
						ContentType:  "application/x-protobuf",
						Body:         data,
						DeliveryMode: amqp.Transient,
						Timestamp:    time.Now(),
						MessageId:    batch.BatchId,
					})

				if err != nil {
					errChan <- fmt.Errorf("worker %d: publish error: %w", workerID, err)
				} else {
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
