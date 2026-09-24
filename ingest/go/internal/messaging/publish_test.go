package messaging

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
)

// TestPublish_StatusCodeContract: non-2xx responses must surface as errors, not count as sent.
func TestPublish_StatusCodeContract(t *testing.T) {
	tests := []struct {
		status  int
		wantErr bool
	}{
		{http.StatusOK, false},
		{http.StatusAccepted, false},
		{http.StatusNoContent, false},
		{http.StatusBadRequest, true},
		{http.StatusUnsupportedMediaType, true},
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusServiceUnavailable, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			cfg := testConfig(srv.URL)
			ps := newTestPubSub(t, cfg, srv.Client())
			batch, data, encoding := testPayload(t, cfg, 10)

			err := ps.doPublish(batch, data, encoding)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("status %d returned nil error — the batch was dropped and reported as delivered", tt.status)
				}
				if !strings.Contains(err.Error(), fmt.Sprint(tt.status)) {
					t.Errorf("error %q does not name status %d", err, tt.status)
				}
			} else if err != nil {
				t.Fatalf("status %d returned error: %v", tt.status, err)
			}
		})
	}
}

// TestPublish_RetriesThenSucceeds covers the flaky-link case: transient
// backpressure must not lose a batch, and the retry must not double-insert.
func TestPublish_RetriesThenSucceeds(t *testing.T) {
	var (
		attempts atomic.Int32
		mu       sync.Mutex
		seenIDs  []string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)

		body, _ := io.ReadAll(r.Body)
		decoded := mustDecode(t, r.Header.Get("Content-Encoding"), body)
		var b TelemetryBatch
		if err := proto.Unmarshal(decoded, &b); err != nil {
			t.Errorf("attempt %d: unmarshal: %v", n, err)
		}
		mu.Lock()
		seenIDs = append(seenIDs, b.BatchId)
		mu.Unlock()

		if n <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.MaxRetries = 3
	ps := newTestPubSub(t, cfg, srv.Client())
	batch, data, encoding := testPayload(t, cfg, 10)

	if err := ps.doPublish(batch, data, encoding); err != nil {
		t.Fatalf("publish should have succeeded on the third attempt: %v", err)
	}
	if got := attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}

	// The batch id must be stable across attempts so a retry is auditable as the same batch.
	mu.Lock()
	defer mu.Unlock()
	if len(seenIDs) != 3 {
		t.Fatalf("server saw %d bodies, want 3", len(seenIDs))
	}
	for i, id := range seenIDs {
		if id != seenIDs[0] {
			t.Errorf("attempt %d batch_id = %q, want %q (stable across retries)", i, id, seenIDs[0])
		}
	}

	if got := ps.failedBatchCount.Load(); got != 0 {
		t.Errorf("failedBatchCount = %d, want 0 — the batch was ultimately delivered", got)
	}
	if got := ps.consecutiveFailures.Load(); got != 0 {
		t.Errorf("consecutiveFailures = %d, want 0 after success", got)
	}
}

// failedBatchCount counts undelivered batches, not attempts.
func TestPublish_RetryExhaustionReturnsError(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.MaxRetries = 2
	ps := newTestPubSub(t, cfg, srv.Client())
	batch, data, encoding := testPayload(t, cfg, 10)

	err := ps.doPublish(batch, data, encoding)
	if err == nil {
		t.Fatal("exhausted retries returned nil error")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("error %q does not name the status", err)
	}
	if !strings.Contains(err.Error(), batch.BatchId) {
		t.Errorf("error %q does not name the batch", err)
	}

	// MaxRetries counts retries after the first attempt.
	if got := attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3 (1 initial + 2 retries)", got)
	}
	if got := ps.failedBatchCount.Load(); got != 1 {
		t.Errorf("failedBatchCount = %d, want 1 — one batch was lost, not %d", got, got)
	}
}

// A 400 (e.g. a compressing producer against an old server) won't fix itself on retry.
func TestPublish_DoesNotRetryBadRequest(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`invalid protobuf body`))
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.MaxRetries = 3
	ps := newTestPubSub(t, cfg, srv.Client())
	batch, data, encoding := testPayload(t, cfg, 10)

	if err := ps.doPublish(batch, data, encoding); err == nil {
		t.Fatal("400 returned nil error")
	}
	if got := attempts.Load(); got != 1 {
		t.Errorf("attempts = %d, want 1 — a 400 is not retryable, the payload cannot fix itself", got)
	}
}

func TestPublish_TransportErrorRetries(t *testing.T) {
	counting := &countingTransport{inner: http.DefaultTransport}
	client := &http.Client{Transport: counting, Timeout: 2 * time.Second}

	// Nothing listens on port 1, so every attempt fails at the transport layer.
	cfg := testConfig("http://127.0.0.1:1/api/ingest")
	cfg.MaxRetries = 2
	ps := newTestPubSub(t, cfg, client)
	batch, data, encoding := testPayload(t, cfg, 10)

	if err := ps.doPublish(batch, data, encoding); err == nil {
		t.Fatal("unreachable server returned nil error")
	}
	if got := counting.calls.Load(); got != 3 {
		t.Errorf("round trips = %d, want 3 (1 initial + 2 retries)", got)
	}
}
