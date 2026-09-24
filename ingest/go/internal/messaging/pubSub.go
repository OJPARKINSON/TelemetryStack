package messaging

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/metrics"
	"github.com/OJPARKINSON/ibt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	activePublishers  sync.WaitGroup
	publisherShutdown atomic.Bool
)

type PubSub struct {
	client      *http.Client
	sessionID   string
	sessionTime time.Time
	config      *config.Config
	workerID    int
	ctx         context.Context
	cancel      context.CancelFunc

	recordBatch []*Telemetry

	totalBatches     int
	totalRecords     int
	totalBytes       int64
	lastFlush        time.Time
	batchSizeBytes   int
	batchSizeRecords int

	// Data persistence for failures
	failedBatchCount   atomic.Int32
	persistedBatches   int
	maxPersistentBytes int64

	// Publish failures fallback
	consecutiveFailures    atomic.Int32
	lastFailureTime        time.Time
	maxConsecutiveFailures int

	// Async publishing
	publishQueue   chan *publishRequest
	publishWg      sync.WaitGroup
	publishDone    chan struct{}
	isShuttingDown atomic.Bool

	mu        sync.Mutex
	closeOnce sync.Once
}

type publishRequest struct {
	batch    *TelemetryBatch
	data     []byte
	encoding string
	errCh    chan error
}

type PublishMetrics struct {
	TotalBatches        int
	TotalRecords        int
	TotalBytes          int64
	CurrentBatchSize    int
	LastFlush           time.Time
	FailedBatches       int32
	PersistedBatches    int
	CircuitBreakerOpen  bool
	ConsecutiveFailures int32
}

func NewPubSub(sessionId string, sessionTime time.Time, cfg *config.Config, client *http.Client, workerId int) *PubSub {
	// Cancelled by Close once the shutdown timeout expires, aborting in-flight requests and retry backoff.
	ctx, cancel := context.WithCancel(context.Background())

	ps := &PubSub{
		sessionID:          sessionId,
		sessionTime:        sessionTime,
		config:             cfg,
		ctx:                ctx,
		cancel:             cancel,
		batchSizeBytes:     cfg.BatchSizeBytes,
		batchSizeRecords:   cfg.BatchSizeRecords,
		lastFlush:          time.Now(),
		maxPersistentBytes: 500 * 1024 * 1024, // 500MB max persistent storage per worker

		consecutiveFailures:    atomic.Int32{},
		maxConsecutiveFailures: 3, // Open circuit after 3 consecutive failures

		// Async publishing - buffer up to 20 batches to prevent blocking
		publishQueue: make(chan *publishRequest, 20),
		publishDone:  make(chan struct{}),

		workerID: workerId,
		client:   client,
	}

	ps.recordBatch = make([]*Telemetry, 0, cfg.BatchSizeRecords)

	// Start async publisher goroutine
	activePublishers.Add(1)
	ps.publishWg.Add(1)
	go ps.publishWorker()

	return ps
}

func (ps *PubSub) recordFailure() {
	ps.consecutiveFailures.Add(1)
	ps.lastFailureTime = time.Now()
}

func (ps *PubSub) recordSuccess() {
	ps.consecutiveFailures.Store(0)
}

// AddStructRecords converts TelemetryTick structs directly to protobuf Telemetry
// records and adds them to the batch, bypassing the map-based path.
func (ps *PubSub) AddStructRecords(ticks []*ibt.TelemetryTick) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	for _, tick := range ticks {
		tickTime := ps.sessionTime.Add(time.Duration(tick.SessionTime * float64(time.Second)))

		record := &Telemetry{
			LapId:              fmt.Sprintf("%d", tick.LapID),
			Speed:              tick.Speed,
			LapDistPct:         tick.LapDistPct,
			SessionNum:         strconv.Itoa(int(tick.SessionNum)),
			SessionType:        tick.SessionType,
			SessionName:        tick.SessionName,
			SessionTime:        tick.SessionTime,
			TrackName:          strings.ReplaceAll(tick.TrackName, " ", "-"),
			TrackId:            strconv.Itoa(tick.TrackID),
			SteeringWheelAngle: tick.SteeringWheelAngle,
			PlayerCarPosition:  tick.PlayerCarPosition,
			VelocityX:          tick.VelocityX,
			VelocityY:          tick.VelocityY,
			VelocityZ:          tick.VelocityZ,
			FuelLevel:          tick.FuelLevel,
			Throttle:           tick.Throttle,
			Brake:              tick.Brake,
			Rpm:                tick.RPM,
			Lat:                tick.Lat,
			Lon:                tick.Lon,
			Gear:               tick.Gear,
			Alt:                tick.Alt,
			LatAccel:           tick.LatAccel,
			LongAccel:          tick.LongAccel,
			VertAccel:          tick.VertAccel,
			Pitch:              tick.Pitch,
			Roll:               tick.Roll,
			Yaw:                tick.Yaw,
			YawNorth:           tick.YawNorth,
			Voltage:            tick.Voltage,
			LapLastLapTime:     tick.LapLastLapTime,
			WaterTemp:          tick.WaterTemp,
			LapDeltaToBestLap:  tick.LapDeltaToBestLap,
			LapCurrentLapTime:  tick.LapCurrentLapTime,
			LFpressure:         tick.LFpressure,
			RFpressure:         tick.RFpressure,
			LRpressure:         tick.LRpressure,
			RRpressure:         tick.RRpressure,
			LFtempM:            tick.LFtempM,
			RFtempM:            tick.RFtempM,
			LRtempM:            tick.LRtempM,
			RRtempM:            tick.RRtempM,
			TickTime:           timestamppb.New(tickTime.UTC()),
		}

		ps.recordBatch = append(ps.recordBatch, record)
		ps.totalRecords++

		estimatedSize := proto.Size(record)
		ps.totalBytes += int64(estimatedSize)
	}

	shouldFlush := len(ps.recordBatch) >= ps.batchSizeRecords ||
		ps.totalBytes >= int64(ps.batchSizeBytes) ||
		time.Since(ps.lastFlush) > time.Duration(ps.config.BatchTimeout)

	if shouldFlush {
		return ps.flushBatchInternal()
	}

	return nil
}

// publishWorker runs in background goroutine to handle async publishing
func (ps *PubSub) publishWorker() {
	defer ps.publishWg.Done()
	defer activePublishers.Done()
	log.Printf("Worker %d: publishWorker goroutine started for session %s", ps.workerID, ps.sessionID)

	for {
		select {
		case req := <-ps.publishQueue:
			log.Printf("Worker %d: Processing batch %s from async queue", ps.workerID, req.batch.BatchId)
			if !ps.config.DryRun {
				err := ps.doPublish(req.batch, req.data, req.encoding)
				if err != nil {
					log.Printf("Worker %d: ERROR publishing batch %s asynchronously: %v",
						ps.workerID, req.batch.BatchId, err)
				} else {
					log.Printf("Worker %d: Successfully published batch %s", ps.workerID, req.batch.BatchId)
				}
				req.errCh <- err
			}
		case <-ps.publishDone:
			log.Printf("Worker %d: Draining %d remaining batches from queue", ps.workerID, len(ps.publishQueue))

			for len(ps.publishQueue) > 0 {
				req := <-ps.publishQueue
				if !ps.config.DryRun {
					err := ps.doPublish(req.batch, req.data, req.encoding)
					if err != nil {
						log.Printf("Worker %d: ERROR publishing batch %s during shutdown: %v",
							ps.workerID, req.batch.BatchId, err)
					}
					req.errCh <- err
				}
			}
			return
		}
	}
}

// doPublish sends a marshalled, compressed batch, retrying 429s and 5xx with exponential backoff.
func (ps *PubSub) doPublish(batch *TelemetryBatch, data []byte, encoding string) error {
	attempts := ps.config.MaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error

	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			delay := ps.config.RetryDelay << (attempt - 1)
			select {
			case <-time.After(delay):
			case <-ps.ctx.Done():
				ps.recordFailure()
				ps.failedBatchCount.Add(1)
				return fmt.Errorf("batch %s: publish cancelled after %d attempts: %w",
					batch.BatchId, attempt, ps.ctx.Err())
			}
		}

		status, err := ps.attemptPublish(data, encoding)

		switch {
		case err == nil && status >= 200 && status < 300:
			ps.recordSuccess()
			return nil

		case err != nil:
			lastErr = err

		default:
			lastErr = fmt.Errorf("server returned %d %s", status, http.StatusText(status))
			if !retryableStatus(status) {
				ps.recordFailure()
				ps.failedBatchCount.Add(1)
				return fmt.Errorf("batch %s: %w", batch.BatchId, lastErr)
			}
		}
	}

	ps.recordFailure()
	ps.failedBatchCount.Add(1)
	return fmt.Errorf("batch %s: publish failed after %d attempts: %w", batch.BatchId, attempts, lastErr)
}

func (ps *PubSub) attemptPublish(data []byte, encoding string) (int, error) {
	req, err := http.NewRequestWithContext(ps.ctx, http.MethodPost, ps.config.IngestURL, bytes.NewReader(data))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	if encoding != "" {
		req.Header.Set("Content-Encoding", encoding)
	}
	req.ContentLength = int64(len(data))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(data)), nil
	}

	res, err := ps.client.Do(req)
	if err != nil {
		return 0, err
	}

	// Drain before closing: Go only reuses a connection once its body is consumed.
	// The limit caps how much a misbehaving server can make us read.
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10))
	res.Body.Close()

	return res.StatusCode, nil
}

// Other 4xx responses won't succeed on resend, so they aren't retried.
func retryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
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

	payload, encoding, err := compressBatch(ps.config, data)
	if err != nil {
		return fmt.Errorf("failed to compress batch %s: %w", batch.BatchId, err)
	}

	metrics.BatchBytesUncompressed.Add(float64(len(data)))
	metrics.BatchBytesWire.Add(float64(len(payload)))
	metrics.BatchSizeBytes.Observe(float64(len(payload)))
	metrics.BatchesSentTotal.Inc()

	advance := func() {
		ps.recordBatch = ps.recordBatch[:0]
		ps.totalBytes = 0
		ps.totalBatches++
		ps.lastFlush = time.Now()
	}

	// During shutdown, publish synchronously to avoid queuing delays
	if ps.isShuttingDown.Load() {
		err := ps.doPublish(batch, payload, encoding)
		advance()
		return err
	}

	req := &publishRequest{
		batch:    batch,
		data:     payload,
		encoding: encoding,
		errCh:    make(chan error, 1),
	}

	// Blocks rather than publishing inline, so two batches from one session are never in flight at once.
	select {
	case ps.publishQueue <- req:
		advance()
		return nil

	case <-ps.publishDone:
		err := ps.doPublish(batch, payload, encoding)
		advance()
		return err
	}
}

func (ps *PubSub) FlushBatch() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.flushBatchInternal()
}

func (ps *PubSub) Close() error {
	var err error
	ps.closeOnce.Do(func() { err = ps.close() })
	return err
}

func (ps *PubSub) close() error {
	// Makes further flushes publish synchronously instead of via the queue.
	ps.isShuttingDown.Store(true)

	// Flush any remaining batches
	if !ps.config.DryRun {
		if err := ps.FlushBatch(); err != nil {
			log.Printf("Worker %d: Error flushing final batch: %v", ps.workerID, err)
		}
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
	case <-time.After(ps.shutdownTimeout()):
		log.Printf("Worker %d: shutdown timed out, abandoning %d queued batches for session %s; re-ingest the file",
			ps.workerID, len(ps.publishQueue), ps.sessionID)
	}

	ps.cancel()

	// Close completes silently - stats available via GetMetrics()
	return nil
}

func (ps *PubSub) shutdownTimeout() time.Duration {
	if ps.config.ShutdownTimeout > 0 {
		return ps.config.ShutdownTimeout
	}
	return 30 * time.Second
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
		FailedBatches:       ps.failedBatchCount.Load(),
		PersistedBatches:    ps.persistedBatches,
		ConsecutiveFailures: ps.consecutiveFailures.Load(),
	}
}

func (ps *PubSub) GetDisplayMetrics() map[string]interface{} {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	return map[string]interface{}{
		"batches_sent":   ps.totalBatches,
		"records_send":   ps.totalRecords,
		"queue_size":     len(ps.publishQueue),
		"failed_batches": ps.failedBatchCount.Load(),
	}
}

func WaitForAllPublishers() {
	publisherShutdown.Store(true)
	log.Println("Waiting for all publishers to finish draining...")
	activePublishers.Wait() // Wait indefinitely until all done
	log.Println("All publishers finished draining")
}
