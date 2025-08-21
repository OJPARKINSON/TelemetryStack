package messaging

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var jsonBufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

type ConnectionPool struct {
	connections []*amqp.Connection
	channels    []*amqp.Channel
	mu          sync.RWMutex
	url         string
	poolSize    int
	current     int
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
			return nil, fmt.Errorf("failed to create connection %d: %w", i, err)
		}

		ch, err := conn.Channel()
		if err != nil {
			conn.Close()
			pool.Close()

			return nil, fmt.Errorf("failed to create channel %d: %w", i, err)
		}

		err = ch.Qos(1000, 0, false)
		if err != nil {
			ch.Close()
			conn.Close()
			pool.Close()
			return nil, fmt.Errorf("failed to set Qos for channel %d: %w", i, err)
		}

		pool.connections[i] = conn
		pool.channels[i] = ch
	}

	return pool, nil
}

func (p *ConnectionPool) GetChannel() *amqp.Channel {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.channels == nil || len(p.channels) == 0 {
		return nil
	}

	ch := p.channels[p.current]
	p.current = (p.current + 1) % p.poolSize

	// Check if channel is still open
	if ch == nil || ch.IsClosed() {
		log.Printf("Channel %d is closed, attempting to recreate", p.current-1)
		return nil
	}

	return ch
}

func (p *ConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := 0; i < len(p.channels); i++ {
		if p.channels[i] != nil {
			p.channels[i].Close()
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
	persistenceDir     string
	failedBatchCount   int
	persistedBatches   int
	maxPersistentBytes int64

	// Circuit breaker for RabbitMQ failures
	circuitBreakerOpen     bool
	consecutiveFailures    int
	lastFailureTime        time.Time
	circuitBreakerTimeout  time.Duration
	maxConsecutiveFailures int

	mu sync.Mutex
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

func NewPubSub(sessionId string, sessionTime time.Time, cfg *config.Config, pool *ConnectionPool) *PubSub {
	// Create persistence directory for failed batches
	persistenceDir := filepath.Join("./failed_batches", sessionId)
	if err := os.MkdirAll(persistenceDir, 0755); err != nil {
		log.Printf("Failed to create persistence directory %s: %v", persistenceDir, err)
		persistenceDir = "" // Disable persistence if directory creation fails
	}

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
		persistenceDir:     persistenceDir,
		maxPersistentBytes: 500 * 1024 * 1024, // 500MB max persistent storage per worker

		// Circuit breaker configuration
		circuitBreakerOpen:     false,
		consecutiveFailures:    0,
		circuitBreakerTimeout:  30 * time.Second, // 30 seconds before retrying RabbitMQ
		maxConsecutiveFailures: 3,                // Open circuit after 3 consecutive failures
	}

	ps.recordBatch = make([]*Telemetry, 0, cfg.BatchSizeRecords)

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

	if len(data) > 0 {
		if wid, ok := data[0]["workerID"]; ok {
			if widInt, ok := wid.(int); ok {
				ps.workerID = widInt
			}
		}
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
	if ps.persistenceDir == "" {
		return fmt.Errorf("persistence disabled - directory creation failed")
	}

	// Check if we've exceeded max persistent storage
	if int64(len(batchData)) > ps.maxPersistentBytes {
		return fmt.Errorf("batch too large for persistence: %d bytes", len(batchData))
	}

	filename := fmt.Sprintf("batch_%s_%d.pb", batchId, time.Now().UnixNano())
	filepath := filepath.Join(ps.persistenceDir, filename)

	if err := os.WriteFile(filepath, batchData, 0644); err != nil {
		return fmt.Errorf("failed to persist batch to %s: %w", filepath, err)
	}

	ps.persistedBatches++
	log.Printf("Worker %d: Persisted batch %s to disk (%d bytes)", ps.workerID, batchId, len(batchData))
	return nil
}

// cleanupOldBatches removes old persisted batches to prevent disk overflow
func (ps *PubSub) cleanupOldBatches() {
	if ps.persistenceDir == "" {
		return
	}

	// Remove batch files older than 24 hours
	cutoff := time.Now().Add(-24 * time.Hour)

	files, err := os.ReadDir(ps.persistenceDir)
	if err != nil {
		log.Printf("Worker %d: Failed to read persistence directory: %v", ps.workerID, err)
		return
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".pb") {
			filepath := filepath.Join(ps.persistenceDir, file.Name())
			info, err := file.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoff) {
				if err := os.Remove(filepath); err != nil {
					log.Printf("Worker %d: Failed to cleanup old batch %s: %v", ps.workerID, file.Name(), err)
				}
			}
		}
	}
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
		sessionNum = fmt.Sprintf("%v", val)
	}

	sessionType := ""
	if val, ok := record["sessionType"]; ok {
		sessionType = fmt.Sprintf("%v", val)
	}

	sessionName := ""
	if val, ok := record["sessionName"]; ok {
		sessionName = fmt.Sprintf("%v", val)
	}

	trackName := ""
	if val, ok := record["trackDisplayShortName"]; ok {
		trackName = fmt.Sprintf("%v", val)
		trackName = strings.ReplaceAll(trackName, " ", "-")
	}

	trackID := ""
	if val, ok := record["trackID"]; ok {
		trackID = fmt.Sprintf("%v", val)
	}

	carID := ""
	if val, ok := record["PlayerCarIdx"]; ok {
		carID = fmt.Sprintf("%v", val)
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
		return fmt.Errorf("failed to marshal protobuf batch: %w", err)
	}

	// Check circuit breaker - if open, skip RabbitMQ and go straight to persistence
	if ps.shouldSkipRabbitMQ() {
		log.Printf("Worker %d: Circuit breaker open, persisting batch %s directly", ps.workerID, batch.BatchId)
		if persistErr := ps.persistBatch(data, batch.BatchId); persistErr != nil {
			return fmt.Errorf("circuit breaker open AND failed to persist batch: %v", persistErr)
		}

		// Mark batch as handled
		ps.recordBatch = ps.recordBatch[:0]
		ps.totalBytes = 0
		ps.totalBatches++
		ps.failedBatchCount++
		ps.lastFlush = time.Now()
		return nil
	}

	maxRetries := 3
	for retry := 0; retry < maxRetries; retry++ {
		ch := ps.pool.GetChannel()
		if ch == nil {
			if retry < maxRetries-1 {
				time.Sleep(time.Duration(retry+1) * 100 * time.Millisecond)
				continue
			}
			return fmt.Errorf("failed to get channel after %d retries", maxRetries)
		}

		ctx, cancel := context.WithTimeout(ps.ctx, 10*time.Second)

		err := ch.PublishWithContext(ctx, "telemetry_topic", "telemetry.ticks", false, false,
			amqp.Publishing{
				ContentType:  "application/x-protobuf",
				Body:         data,
				DeliveryMode: amqp.Transient,
				Timestamp:    time.Now(),
				MessageId:    batch.BatchId,
				Headers: amqp.Table{
					"worker_id":    ps.workerID,
					"record_count": len(ps.recordBatch),
					"batch_size":   len(data),
					"format":       "protobuf",
				},
			})

		cancel()

		if err == nil {
			// Success! Record this and reset circuit breaker
			ps.recordRabbitMQSuccess()

			ps.recordBatch = ps.recordBatch[:0]
			ps.totalBytes = 0
			ps.totalBatches++
			ps.lastFlush = time.Now()

			if ps.totalBatches%50 == 0 {
				log.Printf("Worker %d: Published batch %d (%d records, %d bytes)",
					ps.workerID, ps.totalBatches, len(batch.Records), len(batch.Records))
			}

			return nil
		}

		log.Printf("Worker %d: Failed to publish batch (attempt %d/%d): %v",
			ps.workerID, retry+1, maxRetries, err)

		if retry < maxRetries-1 {
			time.Sleep(time.Duration(retry+1) * 250 * time.Millisecond)
		}
	}

	// If we reach here, RabbitMQ publish failed completely
	// Record the failure for circuit breaker
	ps.recordRabbitMQFailure()

	// Persist the batch to disk for later recovery
	if persistErr := ps.persistBatch(data, batch.BatchId); persistErr != nil {
		log.Printf("Worker %d: CRITICAL - Failed to persist batch %s: %v", ps.workerID, batch.BatchId, persistErr)
		return fmt.Errorf("failed to publish batch after %d retries AND failed to persist: %v", maxRetries, persistErr)
	}

	// Mark batch as handled (persisted) and continue processing
	ps.recordBatch = ps.recordBatch[:0]
	ps.totalBytes = 0
	ps.totalBatches++
	ps.failedBatchCount++
	ps.lastFlush = time.Now()

	log.Printf("Worker %d: Batch %s persisted to disk after RabbitMQ failure (consecutive failures: %d)",
		ps.workerID, batch.BatchId, ps.consecutiveFailures)

	// Periodically clean up old batches
	if ps.failedBatchCount%10 == 0 {
		go ps.cleanupOldBatches()
	}

	return nil // Don't return error since we've handled it via persistence
}

func (ps *PubSub) FlushBatch() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.flushBatchInternal()
}

func (ps *PubSub) Close() error {
	if err := ps.FlushBatch(); err != nil {
		log.Printf("Worker %d: Error flushing final batch: %v", ps.workerID, err)
		return err
	}

	log.Printf("Worker %d: Published %d batches with %d total records", ps.workerID, ps.totalBatches, ps.totalRecords)
	if ps.failedBatchCount > 0 {
		log.Printf("Worker %d: PERSISTENCE - %d batches persisted to disk due to RabbitMQ failures", ps.workerID, ps.persistedBatches)
		log.Printf("Worker %d: Check %s for persisted batches", ps.workerID, ps.persistenceDir)
	}
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
