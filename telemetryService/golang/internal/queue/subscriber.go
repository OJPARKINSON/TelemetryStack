package queue

import (
	"fmt"
	"log"
	"time"

	"github.com/ojparkinson/telemetryService/internal/config"
	"github.com/ojparkinson/telemetryService/internal/messaging"
	"github.com/ojparkinson/telemetryService/internal/metrics"
	"github.com/ojparkinson/telemetryService/internal/persistance"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

type Subscriber struct {
	senderPool *persistance.SenderPool
	stopChan   chan struct{}
}

func NewSubscriber(pool *persistance.SenderPool) *Subscriber {
	return &Subscriber{
		senderPool: pool,
		stopChan:   make(chan struct{}),
	}
}

type batchItem struct {
	batch       *messaging.TelemetryBatch
	deliveryTag uint64
}

func (m *Subscriber) Subscribe(config *config.Config) {
	var conn *amqp.Connection
	var err error

	maxRetries := 10
	baseDelay := 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		conn, err = amqp.Dial("amqp://admin:changeme@" + config.RabbitMQHost + ":5672")
		if err == nil {
			fmt.Println("Successfully connected to RabbitMQ")
			break
		}

		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<uint(attempt))
			fmt.Printf("RabbitMQ connection failed (attempt %d/%d), retrying in %v: %v\n", attempt+1, maxRetries, delay, err)
			time.Sleep(delay)
		} else {
			failOnError(err, "Failed to connect to RabbitMQ after all retries")
		}
	}

	defer conn.Close()

	channel, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer channel.Close()

	err = channel.Qos(20000, 0, false) // reducing prefetch for mem usage
	failOnError(err, "Failed to bind to queue")

	errs := channel.QueueBind("telemetry_queue",
		"telemetry.ticks",
		"telemetry_topic", false, nil)
	failOnError(errs, "Failed to bind to queue")

	msgs, err := channel.Consume("telemetry_queue", "", false, false, false, false, nil)
	failOnError(err, "Failed to consume queue")

	batchChan := make(chan batchItem, 100)

	go m.processBatches(batchChan, channel)

	for event := range msgs {
		batch := &messaging.TelemetryBatch{}
		err := proto.Unmarshal(event.Body, batch)
		if err != nil {
			fmt.Println("error unmarshalling: ", err)
			err := event.Nack(false, false)
			fmt.Println("Failed to ack failed unmarshall: ", err)
			continue
		}

		fmt.Printf("Received batch: session=%v, records=%d\n", batch.SessionId, len(batch.Records))

		// Update Prometheus metrics
		metrics.RecordsReceivedTotal.Add(float64(len(batch.Records)))

		batchChan <- batchItem{
			batch:       batch,
			deliveryTag: event.DeliveryTag,
		}
	}
}

func (m *Subscriber) processBatches(batchChan chan batchItem, channel *amqp.Channel) {
	const (
		targetBatchSize    = 20              // Accumulate 20 RabbitMQ messages
		maxRecordsPerBatch = 25000           // Max 25K telemetry records
		batchTimeout       = 5 * time.Second // Or timeout after 5s
	)

	var (
		batchBuffer  []batchItem
		pendingItem  *batchItem
		timer        = time.NewTimer(batchTimeout)
		totalRecords = 0
	)
	defer timer.Stop()

	for {
		select {
		case <-m.stopChan:
			// Flush remaining batches on shutdown
			if len(batchBuffer) > 0 {
				m.flushBatches(batchBuffer, channel)
			}
			return

		case item := <-batchChan:
			// Handle pending item from previous flush
			if pendingItem != nil {
				batchBuffer = append(batchBuffer, *pendingItem)
				totalRecords += len(pendingItem.batch.Records)
				pendingItem = nil
			}

			// Check if adding this would exceed record limit
			newRecordCount := totalRecords + len(item.batch.Records)
			if newRecordCount > maxRecordsPerBatch && len(batchBuffer) > 0 {
				// Flush current buffer first
				m.flushBatches(batchBuffer, channel)
				batchBuffer = nil
				totalRecords = 0

				// Save this item for next batch
				pendingItem = &item
				timer.Reset(batchTimeout)
				continue
			}

			// Add to buffer
			batchBuffer = append(batchBuffer, item)
			totalRecords += len(item.batch.Records)

			// Flush if we hit target batch size
			if len(batchBuffer) >= targetBatchSize {
				m.flushBatches(batchBuffer, channel)
				batchBuffer = nil
				totalRecords = 0
				timer.Reset(batchTimeout)
			}

		case <-timer.C:
			// Timeout - flush whatever we have
			if len(batchBuffer) > 0 {
				m.flushBatches(batchBuffer, channel)
				batchBuffer = nil
				totalRecords = 0
			}
			timer.Reset(batchTimeout)
		}
	}
}

// collectValidRecords extracts and filters valid telemetry records from batch items
func CollectValidRecords(items []batchItem) []*messaging.Telemetry {
	totalRecords := 0
	for _, item := range items {
		totalRecords += len(item.batch.Records)
	}

	validRecords := make([]*messaging.Telemetry, 0, totalRecords)
	for _, item := range items {
		for _, record := range item.batch.Records {
			if IsValidRecord(record) {
				validRecords = append(validRecords, record)
			}
		}
	}
	return validRecords
}

func (m *Subscriber) flushBatches(items []batchItem, channel *amqp.Channel) {
	validRecords := CollectValidRecords(items)

	if len(validRecords) == 0 {
		for _, item := range items {
			err := channel.Ack(item.deliveryTag, false)

			if err != nil {
				fmt.Println("failed to ack on flush batches", err)
			}
		}
		return
	}

	sender := m.senderPool.Get()
	defer m.senderPool.Return(sender)

	fmt.Printf("Writing %d records from %d messages\n", len(validRecords), len(items))

	// Track write duration
	start := time.Now()
	err := persistance.WriteBatch(sender, validRecords)
	duration := time.Since(start)

	metrics.BatchSizeRecords.Observe(float64(len(validRecords)))
	metrics.DBWriteDuration.Observe(duration.Seconds())

	if err == nil {
		// Success - ACK all messages
		metrics.RecordsWrittenTotal.Add(float64(len(validRecords)))
		fmt.Printf("✅ Successfully wrote %d records, ACKing %d messages \n",
			len(validRecords), len(items))
		for _, item := range items {
			err := channel.Ack(item.deliveryTag, false)

			if err != nil {
				fmt.Println("failed to ack on flush batches", err)
			}
		}
	} else {
		// Failure - NACK all messages for redelivery
		metrics.DBWriteErrors.Inc()
		fmt.Printf("❌ Write failed: %v\n", err)
		fmt.Printf("   NACKing %d messages for redelivery\n", len(items))
		for _, item := range items {
			channel.Nack(item.deliveryTag, false, true) // requeue=true
			fmt.Println("Failed to negatively acknowledge: ", err)
		}
	}
}

func (m *Subscriber) Stop() {
	close(m.stopChan)
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func IsValidRecord(record *messaging.Telemetry) bool {
	return record.SessionId != "" || record.TrackName != ""
}
