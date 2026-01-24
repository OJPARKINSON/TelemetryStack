package messaging

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ConnectionPool struct {
	connections []*amqp.Connection
	channels    []*amqp.Channel
	url         string
	poolSize    int
	current     atomic.Uint32 // Lock-free round-robin counter
	closing     atomic.Bool
}

func NewConnectionPool(url string, poolSize int) (*ConnectionPool, error) {
	pool := &ConnectionPool{
		connections: make([]*amqp.Connection, poolSize),
		channels:    make([]*amqp.Channel, poolSize),
		url:         url,
		poolSize:    poolSize,
	}

	for i := 0; i < poolSize; i++ {
		conn, err := amqp.Dial(url)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create RabbitMQ connection %d to %s: %w\nAction: Verify RabbitMQ is running and credentials are correct", i, url, err)
		}

		ch, err := conn.Channel()
		if err != nil {
			conn.Close()
			pool.Close()

			return nil, fmt.Errorf("failed to create RabbitMQ channel %d: %w\nAction: Check RabbitMQ channel limits and service health", i, err)
		}

		err = ch.Qos(1000, 0, false)
		if err != nil {
			ch.Close()
			conn.Close()
			pool.Close()
			return nil, fmt.Errorf("failed to set QoS for channel %d: %w\nAction: Check RabbitMQ configuration allows prefetch settings", i, err)
		}

		pool.connections[i] = conn
		pool.channels[i] = ch
	}

	return pool, nil
}

func (p *ConnectionPool) GetChannel() *amqp.Channel {
	if p.closing.Load() {
		return nil
	}

	if len(p.channels) == 0 {
		return nil
	}

	// Lock-free round-robin using atomic operations
	idx := p.current.Add(1) % uint32(p.poolSize)
	ch := p.channels[idx]

	// Check if channel is still open
	if ch == nil || ch.IsClosed() {
		log.Printf("Channel %d is closed, attempting to recreate", idx)
		return nil
	}

	return ch
}

func (p *ConnectionPool) Close() {
	p.closing.Store(true)

	time.Sleep(500 * time.Millisecond)

	for i := 0; i < len(p.channels); i++ {
		if p.channels[i] != nil {
			p.channels[i].Close()
		}
	}

	for i := 0; i < len(p.connections); i++ {
		if p.connections[i] != nil {
			p.connections[i].Close()
		}
	}
}

type PubSub struct {
	pool        *ConnectionPool
	sessionID   string
	sessionTime time.Time
	config      *config.Config
	workerID    int
	ctx         context.Context

	recordBatch []*Telemetry
	batchPool   *BatchPool

	totalBatches     int
	totalRecords     int
	totalBytes       int64
	lastFlush        time.Time
	batchSizeBytes   int
	batchSizeRecords int

	// Data persistence for RabbitMQ failures
	failedBatchCount   int
	persistedBatches   int
	maxPersistentBytes int64

	// Circuit breaker for RabbitMQ failures
	circuitBreakerOpen     bool
	consecutiveFailures    int
	lastFailureTime        time.Time
	circuitBreakerTimeout  time.Duration
	maxConsecutiveFailures int

	// Async publishing
	publishQueue   chan *publishRequest
	publishWg      sync.WaitGroup
	publishDone    chan struct{}
	isShuttingDown atomic.Bool

	mu sync.Mutex
}

type publishRequest struct {
	batch *TelemetryBatch
	data  []byte
	errCh chan error
}

type PublishMetrics struct {
	TotalBatches        int
	TotalRecords        int
	TotalBytes          int64
	CurrentBatchSize    int
	LastFlush           time.Time
	FailedBatches       int
	PersistedBatches    int
	CircuitBreakerOpen  bool
	ConsecutiveFailures int
}

func NewPubSub(sessionId string, sessionTime time.Time, cfg *config.Config, pool *ConnectionPool, workerId int) *PubSub {

	ps := &PubSub{
		pool:               pool,
		sessionID:          sessionId,
		sessionTime:        sessionTime,
		config:             cfg,
		ctx:                context.Background(),
		batchPool:          NewBatchPool(cfg.RabbitMQBatchSize),
		batchSizeBytes:     cfg.BatchSizeBytes,
		batchSizeRecords:   cfg.BatchSizeRecords,
		lastFlush:          time.Now(),
		maxPersistentBytes: 500 * 1024 * 1024, // 500MB max persistent storage per worker

		// Circuit breaker configuration
		circuitBreakerOpen:     false,
		consecutiveFailures:    0,
		circuitBreakerTimeout:  30 * time.Second, // 30 seconds before retrying RabbitMQ
		maxConsecutiveFailures: 3,                // Open circuit after 3 consecutive failures

		// Async publishing - buffer up to 20 batches to prevent blocking
		publishQueue: make(chan *publishRequest, 20),
		publishDone:  make(chan struct{}),

		workerID: workerId,
	}

	ps.recordBatch = make([]*Telemetry, 0, cfg.BatchSizeRecords)

	// Start async publisher goroutine
	ps.publishWg.Add(1)
	go ps.publishWorker()

	return ps
}

func getFloatValue(record map[string]interface{}, key string) float64 {
	if val, ok := record[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err == nil {
				return f
			}
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case float32:
			return float64(v)
		}
	}
	return 0.0
}

func getIntValue(record map[string]interface{}, key string) uint32 {
	if val, ok := record[key]; ok {
		switch v := val.(type) {
		case int:
			return uint32(v)
		case int64:
			return uint32(v)
		case float64:
			return uint32(v)
		case string:
			i, err := strconv.Atoi(v)
			if err == nil {
				return uint32(i)
			}
		}
	}
	return 0
}

func (ps *PubSub) Exec(data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	for _, record := range data {
		if err := ps.AddRecord(record); err != nil {
			return fmt.Errorf("failed to add record to batch: %w", err)
		}
	}
	return nil
}

// persistBatch saves a failed batch to disk for later retry
func (ps *PubSub) persistBatch(batchData []byte, batchId string) error {
	return fmt.Errorf("failed to send batch " + batchId)
}

// Circuit breaker methods
func (ps *PubSub) shouldSkipRabbitMQ() bool {
	if !ps.circuitBreakerOpen {
		return false
	}

	// Check if circuit breaker timeout has passed
	if time.Since(ps.lastFailureTime) > ps.circuitBreakerTimeout {
		log.Printf("Worker %d: Circuit breaker timeout reached, attempting to reconnect to RabbitMQ", ps.workerID)
		ps.circuitBreakerOpen = false
		ps.consecutiveFailures = 0
		return false
	}

	return true
}

func (ps *PubSub) recordRabbitMQFailure() {
	ps.consecutiveFailures++
	ps.lastFailureTime = time.Now()

	if ps.consecutiveFailures >= ps.maxConsecutiveFailures {
		if !ps.circuitBreakerOpen {
			log.Printf("Worker %d: CIRCUIT BREAKER OPEN - Too many RabbitMQ failures (%d), switching to persistence-only mode for %v",
				ps.workerID, ps.consecutiveFailures, ps.circuitBreakerTimeout)
		}
		ps.circuitBreakerOpen = true
	}
}

func (ps *PubSub) recordRabbitMQSuccess() {
	if ps.circuitBreakerOpen || ps.consecutiveFailures > 0 {
		log.Printf("Worker %d: RabbitMQ connection recovered, circuit breaker closed", ps.workerID)
	}
	ps.circuitBreakerOpen = false
	ps.consecutiveFailures = 0
}

func (ps *PubSub) AddRecord(record map[string]interface{}) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	tick := ps.transformRecord(record)

	ps.recordBatch = append(ps.recordBatch, tick)
	ps.totalRecords++

	estimatedSize := proto.Size(tick)
	ps.totalBytes += int64(estimatedSize)

	shouldFlush := len(ps.recordBatch) >= ps.batchSizeRecords ||
		ps.totalBytes >= int64(ps.batchSizeBytes) ||
		time.Since(ps.lastFlush) > time.Duration(ps.config.BatchTimeout)

	if shouldFlush {
		return ps.flushBatchInternal()
	}

	return nil
}

func (ps *PubSub) transformRecord(record map[string]interface{}) *Telemetry {
	lapID := getIntValue(record, "Lap")
	sessionTime := getFloatValue(record, "SessionTime")

	sessionNum := ""
	if val, ok := record["SessionNum"]; ok {
		if v, ok := val.(int); ok {
			sessionNum = strconv.Itoa(v)
		}
	}

	sessionType := ""
	if val, ok := record["sessionType"]; ok {
		if v, ok := val.(int); ok {
			sessionType = strconv.Itoa(v)
		}
	}

	sessionName := ""
	if val, ok := record["sessionName"]; ok {
		if v, ok := val.(int); ok {
			sessionName = strconv.Itoa(v)
		}
	}

	trackName := ""
	if val, ok := record["trackDisplayShortName"]; ok {
		trackName = fmt.Sprintf("%v", val)
		trackName = strings.ReplaceAll(trackName, " ", "-")
	}

	trackID := ""
	if val, ok := record["trackID"]; ok {
		if v, ok := val.(int); ok {
			trackID = strconv.Itoa(v)
		}
	}

	carID := ""
	if val, ok := record["PlayerCarIdx"]; ok {
		if v, ok := val.(int); ok {
			carID = strconv.Itoa(v)
		}
	}

	tickTime := ps.sessionTime.Add(time.Duration(sessionTime * float64(time.Second)))

	return &Telemetry{
		LapId:              fmt.Sprintf("%d", lapID),
		Speed:              getFloatValue(record, "Speed"),
		LapDistPct:         getFloatValue(record, "LapDistPct"),
		SessionId:          ps.sessionID,
		SessionNum:         sessionNum,
		SessionType:        sessionType,
		SessionName:        sessionName,
		SessionTime:        sessionTime,
		CarId:              carID,
		TrackName:          trackName,
		TrackId:            trackID,
		WorkerId:           uint32(ps.workerID),
		SteeringWheelAngle: getFloatValue(record, "SteeringWheelAngle"),
		PlayerCarPosition:  getFloatValue(record, "PlayerCarPosition"),
		VelocityX:          getFloatValue(record, "VelocityX"),
		VelocityY:          getFloatValue(record, "VelocityY"),
		VelocityZ:          getFloatValue(record, "VelocityZ"),
		FuelLevel:          getFloatValue(record, "FuelLevel"),
		Throttle:           getFloatValue(record, "Throttle"),
		Brake:              getFloatValue(record, "Brake"),
		Rpm:                getFloatValue(record, "RPM"),
		Lat:                getFloatValue(record, "Lat"),
		Lon:                getFloatValue(record, "Lon"),
		Gear:               getIntValue(record, "Gear"),
		Alt:                getFloatValue(record, "Alt"),
		LatAccel:           getFloatValue(record, "LatAccel"),
		LongAccel:          getFloatValue(record, "LongAccel"),
		VertAccel:          getFloatValue(record, "VertAccel"),
		Pitch:              getFloatValue(record, "Pitch"),
		Roll:               getFloatValue(record, "Roll"),
		Yaw:                getFloatValue(record, "Yaw"),
		YawNorth:           getFloatValue(record, "YawNorth"),
		Voltage:            getFloatValue(record, "Voltage"),
		LapLastLapTime:     getFloatValue(record, "LapLastLapTime"),
		WaterTemp:          getFloatValue(record, "WaterTemp"),
		LapDeltaToBestLap:  getFloatValue(record, "LapDeltaToBestLap"),
		LapCurrentLapTime:  getFloatValue(record, "LapCurrentLapTime"),
		LFpressure:         getFloatValue(record, "LFpressure"),
		RFpressure:         getFloatValue(record, "RFpressure"),
		LRpressure:         getFloatValue(record, "LRpressure"),
		RRpressure:         getFloatValue(record, "RRpressure"),
		LFtempM:            getFloatValue(record, "LFtempM"),
		RFtempM:            getFloatValue(record, "RFtempM"),
		LRtempM:            getFloatValue(record, "LRtempM"),
		RRtempM:            getFloatValue(record, "RRtempM"),
		TickTime:           timestamppb.New(tickTime.UTC()),
	}
}

// publishWorker runs in background goroutine to handle async publishing
func (ps *PubSub) publishWorker() {
	defer ps.publishWg.Done()
	log.Printf("Worker %d: publishWorker goroutine started for session %s", ps.workerID, ps.sessionID)

	for {
		select {
		case req := <-ps.publishQueue:
			log.Printf("Worker %d: Processing batch %s from async queue", ps.workerID, req.batch.BatchId)
			err := ps.doPublish(req.batch, req.data)
			if err != nil {
				log.Printf("Worker %d: ERROR publishing batch %s asynchronously: %v",
					ps.workerID, req.batch.BatchId, err)
			} else {
				log.Printf("Worker %d: Successfully published batch %s", ps.workerID, req.batch.BatchId)
			}
			req.errCh <- err
		case <-ps.publishDone:
			for len(ps.publishQueue) > 0 {
				req := <-ps.publishQueue
				err := ps.doPublish(req.batch, req.data)
				if err != nil {
					log.Printf("Worker %d: ERROR publishing batch %s during shutdown: %v",
						ps.workerID, req.batch.BatchId, err)
				}
				req.errCh <- err
			}
			return
		}
	}
}

// doPublish performs the actual RabbitMQ publish operation
func (ps *PubSub) doPublish(batch *TelemetryBatch, data []byte) error {
	// Check circuit breaker - if open, skip RabbitMQ and go straight to persistence
	if ps.shouldSkipRabbitMQ() {
		log.Printf("Worker %d: Circuit breaker open, persisting batch %s directly", ps.workerID, batch.BatchId)
		if persistErr := ps.persistBatch(data, batch.BatchId); persistErr != nil {
			return fmt.Errorf("circuit breaker open AND failed to persist batch: %v", persistErr)
		}
		return nil
	}

	// During shutdown, only try once with no retries to avoid delays
	maxRetries := 3
	if ps.isShuttingDown.Load() {
		maxRetries = 1
	}

	for retry := 0; retry < maxRetries; retry++ {
		ch := ps.pool.GetChannel()
		if ch == nil {
			if retry < maxRetries-1 {
				time.Sleep(time.Duration(retry+1) * 100 * time.Millisecond)
				continue
			}
			return fmt.Errorf("failed to get RabbitMQ channel after %d retries\nAction: Check RabbitMQ service health and connection pool size", maxRetries)
		}

		// Reduce timeout from 10s to 1s for fast-fail
		ctx, cancel := context.WithTimeout(ps.ctx, 1*time.Second)

		err := ch.PublishWithContext(ctx, "telemetry_topic", "telemetry.ticks", false, false,
			amqp.Publishing{
				ContentType:  "application/x-protobuf",
				Body:         data,
				DeliveryMode: amqp.Transient,
				Timestamp:    time.Now(),
				MessageId:    batch.BatchId,
				Headers: amqp.Table{
					"worker_id":    ps.workerID,
					"record_count": len(batch.Records),
					"batch_size":   len(data),
					"format":       "protobuf",
				},
			})

		cancel()

		if err == nil {
			// Success! Record this and reset circuit breaker
			ps.recordRabbitMQSuccess()
			return nil
		}

		log.Printf("Worker %d: Failed to publish batch (attempt %d/%d): %v",
			ps.workerID, retry+1, maxRetries, err)

		// Skip sleep during shutdown to speed up
		if retry < maxRetries-1 && !ps.isShuttingDown.Load() {
			time.Sleep(time.Duration(retry+1) * 250 * time.Millisecond)
		}
	}

	// If we reach here, RabbitMQ publish failed completely
	// Record the failure for circuit breaker
	ps.recordRabbitMQFailure()

	// Persist the batch to disk for later recovery
	if persistErr := ps.persistBatch(data, batch.BatchId); persistErr != nil {
		log.Printf("Worker %d: CRITICAL - Failed to persist batch %s: %v", ps.workerID, batch.BatchId, persistErr)
		return fmt.Errorf("failed to publish batch after %d retries AND failed to persist: %v\nAction: Check both RabbitMQ connectivity and disk space/permissions", maxRetries, persistErr)
	}

	log.Printf("Worker %d: Batch %s persisted to disk after RabbitMQ failure (consecutive failures: %d)",
		ps.workerID, batch.BatchId, ps.consecutiveFailures)

	// Periodically clean up old batches
	ps.mu.Lock()
	ps.failedBatchCount++
	ps.mu.Unlock()

	return nil // Don't return error since we've handled it via persistence
}

func (ps *PubSub) flushBatchInternal() error {
	if len(ps.recordBatch) == 0 {
		return nil
	}

	batch := &TelemetryBatch{
		Records:   ps.recordBatch,
		BatchId:   fmt.Sprintf("batch_%d_%d_%d", ps.workerID, ps.totalBatches, time.Now().UnixNano()),
		SessionId: ps.sessionID,
		WorkerId:  uint32(ps.workerID),
		Timestamp: timestamppb.New(time.Now()),
	}

	data, err := proto.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf batch: %w\nAction: This is an internal error - check telemetry data validity", err)
	}

	// During shutdown, publish synchronously to avoid queuing delays
	if ps.isShuttingDown.Load() {
		err := ps.doPublish(batch, data)
		ps.recordBatch = ps.recordBatch[:0]
		ps.totalBytes = 0
		ps.totalBatches++
		ps.lastFlush = time.Now()
		return err
	}

	// Try async publishing first (non-blocking if queue has space)
	req := &publishRequest{
		batch: batch,
		data:  data,
		errCh: make(chan error, 1),
	}

	select {
	case ps.publishQueue <- req:
		// Successfully queued for async publishing
		// Clear batch immediately so parser can continue
		ps.recordBatch = ps.recordBatch[:0]
		ps.totalBytes = 0
		ps.totalBatches++
		ps.lastFlush = time.Now()

		// Don't wait for result - let it publish async
		// Errors are logged by the async worker
		return nil

	case <-time.After(100 * time.Millisecond):
		// Queue is full/slow - do sync publish to avoid blocking parser too long
		log.Printf("Worker %d: Publish queue full, falling back to sync publish", ps.workerID)
		err := ps.doPublish(batch, data)

		// Clear batch regardless of error (error is handled via persistence)
		ps.recordBatch = ps.recordBatch[:0]
		ps.totalBytes = 0
		ps.totalBatches++
		ps.lastFlush = time.Now()

		return err
	}
}

func (ps *PubSub) FlushBatch() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.flushBatchInternal()
}

func (ps *PubSub) Close() error {
	// Mark as shutting down to skip retries/delays
	ps.isShuttingDown.Store(true)

	// Flush any remaining batches
	if err := ps.FlushBatch(); err != nil {
		log.Printf("Worker %d: Error flushing final batch: %v", ps.workerID, err)
	}

	// Signal async publisher to shut down
	close(ps.publishDone)

	// Wait for async publisher to finish with timeout
	done := make(chan struct{})
	go func() {
		ps.publishWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Normal shutdown completed
	case <-time.After(4 * time.Second):
		// Timeout - queue taking too long, abandon remaining messages
		// Silently continue - messages may be lost
	}

	// Close completes silently - stats available via GetMetrics()
	return nil
}

func (ps *PubSub) GetMetrics() PublishMetrics {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	return PublishMetrics{
		TotalBatches:        ps.totalBatches,
		TotalRecords:        ps.totalRecords,
		TotalBytes:          ps.totalBytes,
		CurrentBatchSize:    len(ps.recordBatch),
		LastFlush:           ps.lastFlush,
		FailedBatches:       ps.failedBatchCount,
		PersistedBatches:    ps.persistedBatches,
		CircuitBreakerOpen:  ps.circuitBreakerOpen,
		ConsecutiveFailures: ps.consecutiveFailures,
	}
}

func (ps *PubSub) GetDisplayMetrics() map[string]interface{} {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	return map[string]interface{}{
		"batches_sent":    ps.totalBatches,
		"records_send":    ps.totalRecords,
		"queue_size":      len(ps.publishQueue),
		"circuit_breaker": ps.circuitBreakerOpen,
		"failed_batches":  ps.failedBatchCount,
	}
}
