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

// fakeSessions is a domain.SessionRepository backed by maps.
type fakeSessions struct {
	domain.SessionRepository // unimplemented methods panic

	mu       sync.Mutex
	tracks   map[string]domain.Track
	lapTrack map[string]string // "session/lap" -> track_id
}

func (f *fakeSessions) UpsertTrack(_ context.Context, t domain.Track) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.tracks == nil {
		f.tracks = map[string]domain.Track{}
	}
	f.tracks[t.TrackID] = t
	return nil
}

func (f *fakeSessions) GetTrack(_ context.Context, trackID string) (domain.Track, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	t, ok := f.tracks[trackID]
	if !ok {
		return domain.Track{}, domain.ErrNotFound
	}
	return t, nil
}

func (f *fakeSessions) GetLapTrackID(_ context.Context, sessionID, lapID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id, ok := f.lapTrack[sessionID+"/"+lapID]
	if !ok {
		return "", domain.ErrNotFound
	}
	return id, nil
}
