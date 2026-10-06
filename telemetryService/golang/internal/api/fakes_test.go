package api

import (
	"context"
	"sync"

	"github.com/ojparkinson/telemetryService/internal/domain"
)

// fakeWriter is a domain.TelemetryWriter that records what it was handed.
type fakeWriter struct {
	mu      sync.Mutex
	batches [][]*domain.TelemetryPoint
	err     error
}

func (f *fakeWriter) WriteBatch(_ context.Context, records []*domain.TelemetryPoint) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.err != nil {
		return f.err
	}
	f.batches = append(f.batches, records)
	return nil
}

func (f *fakeWriter) calls() [][]*domain.TelemetryPoint {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([][]*domain.TelemetryPoint, len(f.batches))
	copy(out, f.batches)
	return out
}
