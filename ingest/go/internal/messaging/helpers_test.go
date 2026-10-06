package messaging

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/testfixtures"
	"google.golang.org/protobuf/proto"
)

func testConfig(url string) *config.Config {
	return &config.Config{
		IngestURL:        url,
		Compression:      "zstd",
		WorkerCount:      4,
		BatchSizeRecords: 1 << 30,
		BatchSizeBytes:   1 << 30,
		BatchTimeout:     time.Hour,
		MaxRetries:       0,
		RetryDelay:       time.Millisecond,
		ShutdownTimeout:  5 * time.Second,
	}
}

func newTestPubSub(t *testing.T, cfg *config.Config, client *http.Client) *PubSub {
	t.Helper()

	ps := NewPubSub("sess-1", testfixtures.Epoch, cfg, client, 0)
	t.Cleanup(func() { _ = ps.Close() }) // Close is idempotent
	return ps
}

func testPayload(t *testing.T, cfg *config.Config, n int) (*TelemetryBatch, []byte, string) {
	t.Helper()

	recs, err := TransformStructBatch(testfixtures.Ticks(n, 1))
	if err != nil {
		t.Fatal(err)
	}
	batch := &TelemetryBatch{Records: recs, BatchId: "batch-under-test", SessionId: "sess-1"}

	raw, err := proto.Marshal(batch)
	if err != nil {
		t.Fatal(err)
	}
	wire, encoding, err := compressBatch(cfg, raw)
	if err != nil {
		t.Fatal(err)
	}
	return batch, wire, encoding
}

type countingTransport struct {
	inner http.RoundTripper
	calls atomic.Int32
}

func (c *countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.calls.Add(1)
	return c.inner.RoundTrip(r)
}
