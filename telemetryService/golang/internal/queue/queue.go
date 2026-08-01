package queue

import (
	"context"
	"errors"
	"log"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/ojparkinson/telemetryService/internal/domain"
	"github.com/ojparkinson/telemetryService/internal/metrics"
)

var ErrQueueFull = errors.New("ingest queue full")

const dayNanos = int64(24 * time.Hour)

type Queue struct {
	routerCh chan []*domain.TelemetryPoint
	workerCh []chan []*domain.TelemetryPoint
	writer   domain.TelemetryWriter
	routerWg sync.WaitGroup
	workerWg sync.WaitGroup
	Workers  int
	maxMerge int
}

func NewQueue(writer domain.TelemetryWriter) *Queue {
	workers := max(runtime.GOMAXPROCS(0), 2)
	if v := os.Getenv("QUEUE_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			workers = n
		}
	}

	maxMerge := 100_000
	if b := os.Getenv("QUEUE_MAX_MERGE"); b != "" {
		if n, err := strconv.Atoi(b); err == nil && n > 0 {
			maxMerge = n
		}
	}
	bufSize := workers * 8

	queue := &Queue{
		routerCh: make(chan []*domain.TelemetryPoint, bufSize),
		workerCh: make([]chan []*domain.TelemetryPoint, workers),
		writer:   writer,
		routerWg: sync.WaitGroup{},
		workerWg: sync.WaitGroup{},
		Workers:  workers,
		maxMerge: maxMerge,
	}

	for i := range queue.workerCh {
		queue.workerCh[i] = make(chan []*domain.TelemetryPoint, 8)
	}

	return queue
}

func dayIndex(timestamp time.Time) int64 {
	return timestamp.UTC().UnixNano() / dayNanos
}

func (q *Queue) routeFor(batch []*domain.TelemetryPoint) int {
	day := dayIndex(batch[0].Timestamp)
	worker := day % int64(q.Workers)
	if worker < 0 {
		worker += int64(q.Workers)
	}

	return int(worker)
}

func (q *Queue) StartRouter() {
	q.routerWg.Add(1)
	go func() {
		defer q.routerWg.Done()
		for batch := range q.routerCh {
			if len(batch) == 0 {
				continue
			}
			worker := q.routeFor(batch)
			q.workerCh[worker] <- batch
			metrics.QueueDepth.Set(float64(len(q.routerCh)))
		}

		for _, workerCh := range q.workerCh {
			close(workerCh)
		}
	}()
}

func (q *Queue) StartWorker(id int) {
	q.workerWg.Add(1)
	go func() {
		defer q.workerWg.Done()
		for batch := range q.workerCh[id] {
			merged := batch
			coalesced := 0
			for len(merged) < q.maxMerge {
				select {
				case extra, ok := <-q.workerCh[id]:
					if !ok {
						goto flush
					}
					merged = append(merged, extra...)
					coalesced++
				default:
					goto flush
				}
			}
		flush:
			sort.Slice(merged, func(a, b int) bool {
				return merged[a].Timestamp.Before(merged[b].Timestamp)
			})
			metrics.QueueBatchesFlushed.Inc()

			if coalesced > 0 {
				metrics.QueueCoalescedBatches.Add(float64(coalesced))
			}
			if err := q.writer.WriteBatch(context.Background(), merged); err != nil {
				log.Printf("queue worker %d: write failed (%d points): %v", id, len(merged), err)
			}
		}
	}()
}

func (q *Queue) WriteBatch(ctx context.Context, records []*domain.TelemetryPoint) error {
	start := time.Now()
	select {
	case q.routerCh <- records:
		metrics.QueueEnqueueDuration.Observe(time.Since(start).Seconds())
		metrics.QueueDepth.Set(float64(len(q.routerCh)))
		return nil
	case <-ctx.Done():
		return ErrQueueFull
	}
}

func (q *Queue) Shutdown() {
	close(q.routerCh)
	q.routerWg.Wait()
	q.workerWg.Wait()
	log.Println("Ingest queue drained and stopped")
}
