package queue

import (
	"fmt"
	"log"
	"time"

	"github.com/ojparkinson/telemetryService/internal/config"
	"github.com/ojparkinson/telemetryService/internal/messaging"
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
	conn, err := amqp.Dial("amqp://admin:changeme@" + config.RabbitMQHost + ":5672")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	channel, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer channel.Close()

	err = channel.Qos(5000, 0, false)
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
			event.Nack(false, false)
			continue
		}

		fmt.Printf("Received batch: session=%v, records=%d\n", batch.SessionId, len(batch.Records))

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

func (m *Subscriber) flushBatches(items []batchItem, channel *amqp.Channel) {
	var allRecords []*messaging.Telemetry
	for _, item := range items {
		allRecords = append(allRecords, item.batch.Records...)
	}

	validRecords := make([]*messaging.Telemetry, 0, len(allRecords))
	for _, record := range allRecords {
		if IsValidRecord(record) {
			validRecords = append(validRecords, record)
		}
	}

	if len(validRecords) == 0 {
		for _, item := range items {
			channel.Ack(item.deliveryTag, false)
		}
		return
	}

	sender := m.senderPool.Get()
	defer m.senderPool.Return(sender)

	fmt.Printf("Writing %d records from %d messages\n", len(validRecords), len(items))
	err := persistance.WriteBatch(sender, validRecords)
	if err == nil {
		// Success - ACK all messages
		fmt.Printf("✅ Successfully wrote %d records, ACKing %d messages\n",
			len(validRecords), len(items))
		for _, item := range items {
			channel.Ack(item.deliveryTag, false)
		}
	} else {
		// Failure - NACK all messages for redelivery
		fmt.Printf("❌ Write failed: %v\n", err)
		fmt.Printf("   NACKing %d messages for redelivery\n", len(items))
		for _, item := range items {
			channel.Nack(item.deliveryTag, false, true) // requeue=true
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
