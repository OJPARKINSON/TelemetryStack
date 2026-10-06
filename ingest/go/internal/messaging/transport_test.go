package messaging

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/testfixtures"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/sync/errgroup"
)

func mustDecode(t *testing.T, encoding string, body []byte) []byte {
	t.Helper()

	if encoding != "zstd" {
		return body
	}
	dec, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dec.Close()

	out, err := dec.DecodeAll(body, nil)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

// connCountingServer counts new TCP connections.
func connCountingServer(t *testing.T, h http.HandlerFunc, newConns *atomic.Int32) *httptest.Server {
	t.Helper()

	srv := httptest.NewUnstartedServer(h)
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			newConns.Add(1)
		}
	}
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

// DefaultTransport keeps only 2 idle conns per host; the shared client must reuse
// connections across workers.
func TestPublish_ReusesConnectionsAcrossWorkers(t *testing.T) {
	const (
		workers = 4
		batches = 10
	)

	var (
		newConns atomic.Int32
		requests atomic.Int32
	)

	srv := connCountingServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}, &newConns)

	cfg := testConfig(srv.URL)
	cfg.WorkerCount = workers
	client := NewHTTPClient(cfg)

	var g errgroup.Group
	for i := 0; i < workers; i++ {
		g.Go(func() error {
			ps := NewPubSub("sess-1", testfixtures.Epoch, cfg, client, 0)
			defer ps.Close()

			batch, data, encoding := testPayload(t, cfg, 50)
			for b := 0; b < batches; b++ {
				if err := ps.doPublish(batch, data, encoding); err != nil {
					return err
				}
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Control: the assertion below is meaningless if no requests happened.
	if got := requests.Load(); got != workers*batches {
		t.Fatalf("requests = %d, want %d", got, workers*batches)
	}

	got := newConns.Load()
	if got > workers {
		t.Errorf("opened %d connections for %d requests across %d workers, want <= %d",
			got, workers*batches, workers, workers)
	}
	// Stated separately so a failure distinguishes "regressed to a connection
	// per request" from "off by one".
	if got >= workers*batches {
		t.Errorf("opened %d connections for %d requests — connections are not being reused at all",
			got, workers*batches)
	}
	t.Logf("%d new connections for %d requests across %d workers", got, workers*batches, workers)
}

// Error responses must be drained so their connections return to the idle pool.
func TestPublish_DrainsResponseBodyForReuse(t *testing.T) {
	var newConns atomic.Int32

	errBody := bytes.Repeat([]byte("x"), 2048)
	srv := connCountingServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(errBody)
	}, &newConns)

	cfg := testConfig(srv.URL)
	cfg.MaxRetries = 0
	ps := newTestPubSub(t, cfg, NewHTTPClient(cfg))
	batch, data, encoding := testPayload(t, cfg, 10)

	for i := 0; i < 5; i++ {
		_ = ps.doPublish(batch, data, encoding) // errors expected; not the point
	}

	if got := newConns.Load(); got != 1 {
		t.Errorf("opened %d connections for 5 sequential requests, want 1 — "+
			"the response body is not being drained, so the connection is discarded each time", got)
	}
}

// Under backpressure a flush must block rather than publish inline, or two batches
// from one session end up in flight with no ordering between them.
func TestPublish_BlocksRatherThanDoubleFlighting(t *testing.T) {
	var (
		inFlight atomic.Int32
		maxSeen  atomic.Int32
		handled  atomic.Int32
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := inFlight.Add(1)
		for {
			old := maxSeen.Load()
			if n <= old || maxSeen.CompareAndSwap(old, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond) // slow enough to fill the queue
		inFlight.Add(-1)
		handled.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.BatchSizeRecords = 25
	ps := newTestPubSub(t, cfg, srv.Client())

	// AddStructRecords checks the flush condition once per call, so feed it in chunks.
	// 30 chunks exceeds the 20-deep publish queue, which is what makes the send block.
	ticks := testfixtures.Ticks(25*30, 1)
	for i := 0; i < len(ticks); i += 25 {
		if err := ps.AddStructRecords(ticks[i : i+25]); err != nil {
			t.Fatalf("AddStructRecords: %v", err)
		}
	}
	if err := ps.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if got := handled.Load(); got < 25 {
		t.Fatalf("server handled %d batches, want ~30 — the test did not exercise the queue", got)
	}
	if got := maxSeen.Load(); got != 1 {
		t.Errorf("max concurrent in-flight requests for one session = %d, want 1", got)
	}
}

var _ = config.Config{}
