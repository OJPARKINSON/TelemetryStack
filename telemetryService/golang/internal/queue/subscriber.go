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

type workItem struct {
	batchItems  []batchItem
	deliverTags []uint64
	resultChan  chan<- workResult
}

type workResult struct {
	deliverTags []uint64
	success     bool
	err         error
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

	err = channel.Qos(100, 0, false) // 10 workers × 10 batches ahead = reasonable prefetch
	failOnError(err, "Failed to set QoS prefetch")

	errs := channel.QueueBind("telemetry_queue",
		"telemetry.ticks",
		"telemetry_topic", false, nil)
	failOnError(errs, "Failed to bind to queue")

	msgs, err := channel.Consume("telemetry_queue", "", false, false, false, false, nil)
	failOnError(err, "Failed to consume queue")

	batchChan := make(chan batchItem, 100)

	receivedCount := 0
	lastReport := time.Now()

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

		// fmt.Printf("Received batch: session=%v, records=%d\n", batch.SessionId, len(batch.Records))
		receivedCount++

		if time.Since(lastReport) > 5*time.Second {
			rate := float64(receivedCount) / time.Since(lastReport).Seconds()
			fmt.Printf("📥 Receiving from RabbitMQ: %.0f batches/sec, batchChan depth: %d/%d\n",
				rate, len(batchChan), cap(batchChan))
			receivedCount = 0
			lastReport = time.Now()
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
		targetBatchSize    = 10 // ← REDUCE from 40 (emergency fix)
		maxRecordsPerBatch = 1000000
		batchTimeout       = 500 * time.Millisecond // ← REDUCE from 1.5s
		numWorkers         = 10
	)

	workChan := make(chan workItem, numWorkers*2)
	resultChan := make(chan workResult, numWorkers*2)

	m.startWorkerPool(workChan, numWorkers)

	go m.resultHandler(resultChan, channel)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var (
		batchBuffer  []batchItem
		timer        *time.Timer
		totalRecords = 0

		batchesSent = 0
	)
	defer timer.Stop()

	for {
		var timeoutChan <-chan time.Time
		if len(batchBuffer) > 0 {
			if timer == nil {
				timer = time.NewTimer(batchTimeout)
			}
			timeoutChan = timer.C
		}

		sendToWorkers := func(items []batchItem) {
			deliveryTags := make([]uint64, len(items))
			for i, item := range items {
				deliveryTags[i] = item.deliveryTag
			}

			workChan <- workItem{
				batchItems:  items,
				deliverTags: deliveryTags, // Fixed typo
				resultChan:  resultChan,
			}

			batchesSent++
		}

		select {
		case <-ticker.C:
			// Report queue depths every 5 seconds
			log.Printf("📊 Queue Status: batchChan=%d/%d, workChan=%d/%d, resultChan=%d/%d, buffered=%d batches (%d records), sent=%d batches",
				len(batchChan), cap(batchChan),
				len(workChan), cap(workChan),
				len(resultChan), cap(resultChan),
				len(batchBuffer), totalRecords,
				batchesSent)
			batchesSent = 0
		case <-m.stopChan:
			if timer != nil {
				timer.Stop()
			}
			if len(batchBuffer) > 0 {
				sendToWorkers(batchBuffer)
			}
			close(workChan) // Signal workers to finish
			return

		case item := <-batchChan:
			newRecordCount := totalRecords + len(item.batch.Records)
			if newRecordCount > maxRecordsPerBatch && len(batchBuffer) > 0 {
				sendToWorkers(batchBuffer)
				batchBuffer = []batchItem{item}
				totalRecords = len(item.batch.Records)
				timer.Reset(batchTimeout)
				continue
			}

			batchBuffer = append(batchBuffer, item)
			totalRecords += len(item.batch.Records)

			if len(batchBuffer) >= targetBatchSize {
				sendToWorkers(batchBuffer)
				batchBuffer = nil
				totalRecords = 0
				timer.Reset(batchTimeout)
			}

		case <-timeoutChan:
			if len(batchBuffer) > 0 {
				sendToWorkers(batchBuffer)
				batchBuffer = nil
				totalRecords = 0
			}
			timer = nil
		}

		// case <-timer.C:
		// 	// Timeout - flush whatever we have
		// 	if len(batchBuffer) > 0 {
		// 		m.flushBatches(batchBuffer, channel)
		// 		batchBuffer = nil
		// 		totalRecords = 0
		// 	}
		// 	timer.Reset(batchTimeout)
		// }
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

func (m *Subscriber) startWorkerPool(workChan <-chan workItem, numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		go m.worker(i, workChan)
	}
}

func (m *Subscriber) worker(id int, workChan <-chan workItem) {
	for work := range workChan {
		sender := m.senderPool.Get()

		validRecords := CollectValidRecords(work.batchItems)

		start := time.Now()
		err := persistance.WriteBatch(sender, validRecords)
		duration := time.Since(start)

		m.senderPool.Return(sender)

		work.resultChan <- workResult{
			deliverTags: work.deliverTags,
			success:     err == nil,
			err:         err,
		}

		metrics.DBWriteDuration.Observe(duration.Seconds())
		if err == nil {
			metrics.RecordsWrittenTotal.Add(float64(len(validRecords)))
			log.Printf("Worker %d: wrote %d records in %v", id, len(validRecords), duration)
		} else {
			metrics.DBWriteErrors.Inc()
			log.Printf("Worker %d: write failed for %d records in %v: %v", id, len(validRecords), duration, err)
		}
	}
}

func (m *Subscriber) resultHandler(resultChan <-chan workResult, channel *amqp.Channel) {
	for result := range resultChan {
		if result.success {
			for _, tag := range result.deliverTags {
				if err := channel.Ack(tag, false); err != nil {
					log.Printf("Failed to ACK tag %d: %v", tag, err)
				}
			}
		} else {
			log.Printf("Write failed: %v, NACKing %d messages", result.err, len(result.deliverTags))
			for _, tag := range result.deliverTags {
				if err := channel.Nack(tag, false, true); err != nil {
					log.Printf("Failed to NACK tag %d: %v", tag, err)
				}
			}

		}
	}
}
