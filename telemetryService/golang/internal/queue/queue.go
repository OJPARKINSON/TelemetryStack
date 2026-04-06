package queue

import (
	"context"
	"errors"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/ojparkinson/telemetryService/internal/domain"
	"github.com/ojparkinson/telemetryService/internal/metrics"
)

var ErrQueueFull = errors.New("ingest queue full")

type Queue struct {
	ch      chan []*domain.TelemetryPoint
	writer  domain.TelemetryWriter
	wg      sync.WaitGroup
	workers int
}

func NewQueue(writer domain.TelemetryWriter) *Queue {
	workers := max(runtime.GOMAXPROCS(0), 2)
	bufSize := workers * 8

	return &Queue{
		ch:      make(chan []*domain.TelemetryPoint, bufSize),
		writer:  writer,
		wg:      sync.WaitGroup{},
		workers: workers,
	}
}

func (q *Queue) Start() {
	for i := range q.workers {
		q.wg.Add(1)
		go func(id int) {
			defer q.wg.Done()
			for batch := range q.ch {
				merged := batch
				coalesced := 0
				for {
					select {
					case extra := <-q.ch:
						merged = append(merged, extra...)
						coalesced++
					default:
						goto flush
					}
				}
			flush:
				metrics.QueueDepth.Set(float64(len(q.ch)))
				metrics.QueueBatchesFlushed.Inc()
				if coalesced > 0 {
					metrics.QueueCoalescedBatches.Add(float64(coalesced))
				}
				if err := q.writer.WriteBatch(context.Background(), merged); err != nil {
					log.Printf("queue worker %d: write failed (%d points): %v", id, len(merged), err)
				}
			}
		}(i)
	}
	log.Printf("Ingest queue started: %d workers, buffer %d", q.workers, cap(q.ch))
}

func (q *Queue) WriteBatch(ctx context.Context, records []*domain.TelemetryPoint) error {
	start := time.Now()
	select {
	case q.ch <- records:
		metrics.QueueEnqueueDuration.Observe(time.Since(start).Seconds())
		metrics.QueueDepth.Set(float64(len(q.ch)))
		return nil
	case <-ctx.Done():
		return ErrQueueFull
	}
}

func (q *Queue) Shutdown() {
	close(q.ch)
	q.wg.Wait()
	log.Println("Ingest queue drained and stopped")
}
