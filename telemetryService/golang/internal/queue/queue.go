package queue

import (
	"context"
	"errors"
	"log"
	"runtime"
	"sync"

	"github.com/ojparkinson/telemetryService/internal/domain"
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
				for {
					select {
					case extra := <-q.ch:
						merged = append(merged, extra...)
					default:
						goto flush
					}
				}
			flush:
				if err := q.writer.WriteBatch(context.Background(), merged); err != nil {
					log.Printf("queue worker %d: write failed (%d points): %v", id, len(merged), err)
				}
			}
		}(i)
	}
	log.Printf("Ingest queue started: %d workers, buffer %d", q.workers, cap(q.ch))
}

func (q *Queue) WriteBatch(ctx context.Context, records []*domain.TelemetryPoint) error {
	select {
	case q.ch <- records:
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
