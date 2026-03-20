package processing

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/messaging"
	"github.com/OJPARKINSON/ibt"
	"github.com/OJPARKINSON/ibt/headers"
)

// loaderProcessor processes telemetry data and sends it to RabbitMQ.
// It uses struct-based processing for optimal performance.
type loaderProcessor struct {
	pubSub         *messaging.PubSub
	cache          []*ibt.TelemetryTick
	groupNumber    int
	thresholdBytes int
	workerID       int
	mu             sync.Mutex
	config         *config.Config

	subSessionID   string // The unique SubSessionID for this group
	sessionMap     map[int]sessionInfo
	trackName      string
	trackID        int
	sessionInfoSet bool

	tickPool *sync.Pool

	// Metrics tracking
	totalProcessed int
	totalBatches   int

	// Progress tracking
	progressCallback ProgressCallback
	currentFile      string
}

// sessionInfo holds session metadata
type sessionInfo struct {
	sessionNum  int
	sessionType string
	sessionName string
}

// ProcessorMetrics contains telemetry processing metrics
type ProcessorMetrics struct {
	TotalProcessed       int
	TotalBatches         int
	ProcessingTime       time.Duration
	MaxBatchSize         int
	ProcessingStarted    time.Time
	MemoryPressureEvents int
	AdaptiveBatchSize    int
}

// NewProcessor creates a new telemetry processor
func NewProcessor(pubSub *messaging.PubSub, groupNumber int, config *config.Config, workerID int, subSessionID string) *loaderProcessor {
	return &loaderProcessor{
		pubSub:           pubSub,
		cache:            make([]*ibt.TelemetryTick, 0, config.BatchSizeRecords),
		groupNumber:      groupNumber,
		config:           config,
		thresholdBytes:   config.BatchSizeBytes,
		workerID:         workerID,
		subSessionID:     subSessionID,
		sessionMap:       make(map[int]sessionInfo),
		progressCallback: &NoOpProgressCallback{},
		tickPool: &sync.Pool{
			New: func() any {
				return &ibt.TelemetryTick{}
			},
		},
	}
}

func (l *loaderProcessor) SetProgressCallback(callback ProgressCallback, filename string) {
	if callback != nil {
		l.progressCallback = callback
		l.currentFile = filename
	}
}

func (l *loaderProcessor) ProcessStruct(tick *ibt.TelemetryTick, hasNext bool, session *headers.Session) error {
	if !l.sessionInfoSet && session != nil && len(session.SessionInfo.Sessions) > 0 {
		for _, sess := range session.SessionInfo.Sessions {
			l.sessionMap[sess.SessionNum] = sessionInfo{
				sessionNum:  sess.SessionNum,
				sessionType: sess.SessionType,
				sessionName: sess.SessionName,
			}
		}
		l.trackName = session.WeekendInfo.TrackDisplayShortName
		l.trackID = session.WeekendInfo.TrackID
		l.sessionInfoSet = true
	}

	tick.GroupNum = l.groupNumber
	tick.WorkerID = l.workerID
	tick.TrackName = l.trackName
	tick.TrackID = l.trackID

	// Use the SubSessionID from the header instead of SessionNum
	tick.SessionID = l.subSessionID

	if sessionInfo, exists := l.sessionMap[int(tick.SessionNum)]; exists {
		tick.SessionType = sessionInfo.sessionType
		tick.SessionName = sessionInfo.sessionName
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	estimatedSize := 512

	shouldFlush := len(l.cache) >= l.config.BatchSizeRecords || len(l.cache)*estimatedSize > l.thresholdBytes

	if shouldFlush && len(l.cache) > 0 {
		if err := l.loadBatch(); err != nil {
			return fmt.Errorf("failed to load batch: %w", err)
		}
	}

	l.cache = append(l.cache, tick)
	l.totalProcessed++

	return nil
}

func (l *loaderProcessor) loadBatch() error {
	if len(l.cache) == 0 {
		return nil
	}

	batchSize := len(l.cache)

	if !l.config.DisableRabbitMQ {
		err := l.pubSub.ExecStructs(l.cache)
		if err != nil {
			return err
		}
	}

	for _, tick := range l.cache {
		l.tickPool.Put(tick)
	}

	l.cache = l.cache[:0]
	l.totalBatches++

	// Report progress after batch is sent
	l.progressCallback.OnBatchSent(l.currentFile, l.totalProcessed, l.totalBatches)

	_ = batchSize // Keep for potential future use

	return nil
}

func (l *loaderProcessor) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.cache) > 0 {
		return l.loadBatch()
	}

	return nil
}

// Fields defines the telemetry fields this processor needs.
// Whitelist is automatically extracted from ibt tags.
func (l *loaderProcessor) Fields() any {
	return struct {
		// Lap & Position
		LapID             int32   `ibt:"Lap"`
		LapDistPct        float64 `ibt:"LapDistPct"`
		Speed             float64 `ibt:"Speed"`
		PlayerCarPosition float64 `ibt:"PlayerCarPosition"`
		PlayerCarIdx      int32   `ibt:"PlayerCarIdx"`

		// Pedals & Gear
		Throttle float64 `ibt:"Throttle"`
		Brake    float64 `ibt:"Brake"`
		Gear     int32   `ibt:"Gear"`
		RPM      float64 `ibt:"RPM"`

		// Steering & Velocity
		SteeringWheelAngle float64 `ibt:"SteeringWheelAngle"`
		VelocityX          float64 `ibt:"VelocityX"`
		VelocityY          float64 `ibt:"VelocityY"`
		VelocityZ          float64 `ibt:"VelocityZ"`

		// GPS & Orientation
		Lat      float64 `ibt:"Lat"`
		Lon      float64 `ibt:"Lon"`
		Alt      float64 `ibt:"alt"`
		Pitch    float64 `ibt:"pitch"`
		Roll     float64 `ibt:"roll"`
		Yaw      float64 `ibt:"yaw"`
		YawNorth float64 `ibt:"YawNorth"`

		// Acceleration
		LatAccel  float64 `ibt:"LatAccel"`
		LongAccel float64 `ibt:"LongAccel"`
		VertAccel float64 `ibt:"VertAccel"`

		// Session & Timing
		SessionTime       float64 `ibt:"SessionTime"`
		SessionNum        int32   `ibt:"SessionNum"`
		FuelLevel         float64 `ibt:"FuelLevel"`
		Voltage           float64 `ibt:"Voltage"`
		WaterTemp         float64 `ibt:"WaterTemp"`
		LapLastLapTime    float64 `ibt:"LapLastLapTime"`
		LapDeltaToBestLap float64 `ibt:"LapDeltaToBestLap"`
		LapCurrentLapTime float64 `ibt:"LapCurrentLapTime"`

		// Tire Pressures
		LFpressure float64 `ibt:"LFpressure"`
		RFpressure float64 `ibt:"RFpressure"`
		LRpressure float64 `ibt:"LRpressure"`
		RRpressure float64 `ibt:"RRpressure"`

		// Tire Temps
		LFtempM float64 `ibt:"LFtempM"`
		RFtempM float64 `ibt:"RFtempM"`
		LRtempM float64 `ibt:"LRtempM"`
		RRtempM float64 `ibt:"RRtempM"`
	}{}
}

func (l *loaderProcessor) FlushPendingData() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.cache) > 0 {
		log.Printf("Worker %d: Flushing %d pending struct records",
			l.workerID, len(l.cache))
		return l.loadBatch()
	}
	return nil
}

func (l *loaderProcessor) GetMetrics() any {
	return nil // Return nil for now - can be extended later if needed
}
