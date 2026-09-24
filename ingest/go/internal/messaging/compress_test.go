package messaging

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/testfixtures"
	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"
)

type capturedRequest struct {
	contentType     string
	contentEncoding string
	encodingPresent bool
	body            []byte
}

func captureServer(t *testing.T, got *capturedRequest) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		got.contentType = r.Header.Get("Content-Type")
		got.contentEncoding = r.Header.Get("Content-Encoding")
		_, got.encodingPresent = r.Header["Content-Encoding"]
		got.body = body
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestPublish_SendsZstdEncodedProtobuf(t *testing.T) {
	const records = 24000

	var got capturedRequest
	srv := captureServer(t, &got)

	cfg := testConfig(srv.URL)
	ps := newTestPubSub(t, cfg, srv.Client())

	if err := ps.AddStructRecords(testfixtures.Ticks(records, 1)); err != nil {
		t.Fatalf("AddStructRecords: %v", err)
	}
	if err := ps.Close(); err != nil { // flushes and drains the publisher
		t.Fatalf("Close: %v", err)
	}

	if got.contentEncoding != "zstd" {
		t.Errorf("Content-Encoding = %q, want %q", got.contentEncoding, "zstd")
	}

	if got.contentType != "application/x-protobuf" {
		t.Errorf("Content-Type = %q, want %q", got.contentType, "application/x-protobuf")
	}

	dec, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dec.Close()

	decoded, err := dec.DecodeAll(got.body, nil)
	if err != nil {
		t.Fatalf("body is not valid zstd: %v", err)
	}

	var batch TelemetryBatch
	if err := proto.Unmarshal(decoded, &batch); err != nil {
		t.Fatalf("decompressed body is not a TelemetryBatch: %v", err)
	}
	if len(batch.Records) != records {
		t.Fatalf("got %d records, want %d", len(batch.Records), records)
	}

	want, err := TransformStructBatch(testfixtures.Ticks(records, 1))
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range []int{0, records / 2, records - 1} {
		if batch.Records[i].Speed != want[i].Speed {
			t.Errorf("record %d Speed = %v, want %v", i, batch.Records[i].Speed, want[i].Speed)
		}
		if !batch.Records[i].TickTime.AsTime().Equal(want[i].TickTime.AsTime()) {
			t.Errorf("record %d TickTime = %v, want %v",
				i, batch.Records[i].TickTime.AsTime(), want[i].TickTime.AsTime())
		}
	}

	t.Logf("wire %d B for %d records (%.1f B/tick)", len(got.body), records, float64(len(got.body))/records)
}

func TestPublish_CompressionDisabledSendsRaw(t *testing.T) {
	var got capturedRequest
	srv := captureServer(t, &got)

	cfg := testConfig(srv.URL)
	cfg.Compression = "none"
	ps := newTestPubSub(t, cfg, srv.Client())

	if err := ps.AddStructRecords(testfixtures.Ticks(100, 1)); err != nil {
		t.Fatalf("AddStructRecords: %v", err)
	}
	if err := ps.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if got.encodingPresent {
		t.Errorf("Content-Encoding = %q, want the header to be absent", got.contentEncoding)
	}

	var batch TelemetryBatch
	if err := proto.Unmarshal(got.body, &batch); err != nil {
		t.Fatalf("body should be raw protobuf: %v", err)
	}
	if len(batch.Records) != 100 {
		t.Errorf("got %d records, want 100", len(batch.Records))
	}
}
