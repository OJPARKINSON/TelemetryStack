package processing

import (
	"fmt"
	"log"
	"sync"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/messaging"
	"github.com/OJPARKINSON/ibt"
	"github.com/OJPARKINSON/ibt/headers"
)

type structLoaderProcessor struct {
	pubSub         *messaging.PubSub
	cache          []*ibt.TelemetryTick
	groupNumber    int
	thresholdBytes int
	workerID       int
	mu             sync.Mutex
	config         *config.Config

	sessionMap     map[int]sessionInfo
	trackName      string
	trackID        int
	sessionInfoSet bool

	tickPool *sync.Pool

	// Metrics tracking
	totalProcessed int
	totalBatches   int
}

func NewStructProcessor(pubSub *messaging.PubSub, groupNumber int, config *config.Config, workerID int) *structLoaderProcessor {
	return &structLoaderProcessor{
		pubSub:         pubSub,
		cache:          make([]*ibt.TelemetryTick, 0, config.BatchSizeRecords),
		groupNumber:    groupNumber,
		config:         config,
		thresholdBytes: config.BatchSizeBytes,
		workerID:       workerID,
		sessionMap:     make(map[int]sessionInfo),
		tickPool: &sync.Pool{
			New: func() interface{} {
				return &ibt.TelemetryTick{}
			},
		},
	}
}

func (l *structLoaderProcessor) ProcessStruct(tick *ibt.TelemetryTick, hasNext bool, session *headers.Session) error {
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

	if sessionInfo, exists := l.sessionMap[int(tick.SessionNum)]; exists {
		tick.SessionID = fmt.Sprintf("%d", sessionInfo.sessionNum)
		tick.SessionType = sessionInfo.sessionType
		tick.SessionName = sessionInfo.sessionName
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	estimatedSize := 512

	shouldFlush := len(l.cache) >= l.config.BatchSizeRecords || len(l.cache)*estimatedSize > l.thresholdBytes

	if shouldFlush && len(l.cache) > 0 {
		if err := l.loadBatch(); err != nil {
			return fmt.Errorf("failed to laod batch: %w", err)
		}
	}

	l.cache = append(l.cache, tick)
	l.totalProcessed++

	return nil
}

func (l *structLoaderProcessor) loadBatch() error {
	if len(l.cache) == 0 {
		return nil
	}

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

	return nil
}

func (l *structLoaderProcessor) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.cache) > 0 {
		log.Printf("Worker %d: Flushing remaining %d struct records on close",
			l.workerID, len(l.cache))
		return l.loadBatch()
	}

	return nil
}

func (l *structLoaderProcessor) Process(input ibt.Tick, hasNext bool, session *headers.Session) error {
	// Stub implementation - struct processor uses ProcessStruct instead
	return fmt.Errorf("Process() not implemented - use ProcessStruct()")
}

func (l *structLoaderProcessor) Whitelist() []string {
	return []string{
		"Lap", "LapDistPct", "Speed", "Throttle", "Brake", "Gear", "RPM",
		"SteeringWheelAngle", "VelocityX", "VelocityY", "VelocityZ", "Lat", "Lon",
		"SessionTime", "PlayerCarPosition", "FuelLevel", "PlayerCarIdx", "SessionNum",
		"alt", "LatAccel", "LongAccel", "VertAccel", "pitch", "roll", "yaw",
		"YawNorth", "Voltage", "LapLastLapTime", "WaterTemp", "LapDeltaToBestLap",
		"LapCurrentLapTime", "LFpressure", "RFpressure", "LRpressure", "RRpressure",
		"LFtempM", "RFtempM", "LRtempM", "RRtempM",
	}
}

func (l *structLoaderProcessor) FlushPendingData() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.cache) > 0 {
		log.Printf("Worker %d: Flushing %d pending struct records",
			l.workerID, len(l.cache))
		return l.loadBatch()
	}
	return nil
}

func (l *structLoaderProcessor) GetMetrics() interface{} {
	l.mu.Lock()
	defer l.mu.Unlock()

	return ProcessorMetrics{
		TotalProcessed: l.totalProcessed,
		TotalBatches:   l.totalBatches,
	}
}
