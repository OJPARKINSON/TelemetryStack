package api

import (
	"bytes"
	"compress/gzip"
	crand "crypto/rand"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/ojparkinson/telemetryService/internal/domain"
	"github.com/ojparkinson/telemetryService/internal/messaging"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// fixedTick is an arbitrary but stable instant, so round-trip assertions can
// compare timestamps exactly.
var fixedTick = time.Date(2025, 6, 10, 17, 25, 58, 0, time.UTC)

func newTestServer(w domain.TelemetryWriter) *Server {
	return &Server{
		writer: w,
		logger: log.New(io.Discard, "", 0),
	}
}

func doIngest(s *Server, body []byte, contentType, contentEncoding string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/ingest", bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if contentEncoding != "" {
		req.Header.Set("Content-Encoding", contentEncoding)
	}

	rec := httptest.NewRecorder()
	s.handleIngest(rec, req)
	return rec
}

func zstdEncodeAll(t *testing.T, b []byte) []byte {
	t.Helper()

	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		t.Fatalf("zstd.NewWriter: %v", err)
	}
	defer enc.Close()

	return enc.EncodeAll(b, nil)
}

func zstdEncodeStream(t *testing.T, b []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		t.Fatalf("zstd.NewWriter: %v", err)
	}
	if _, err := enc.Write(b); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return buf.Bytes()
}

func marshal(t *testing.T, m proto.Message) []byte {
	t.Helper()

	b, err := proto.Marshal(m)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}
	return b
}

func sampleBatch() *messaging.TelemetryBatch {
	return &messaging.TelemetryBatch{
		BatchId:   "batch-1",
		SessionId: "sess-42",
		CarId:     "car-7",
		Records: []*messaging.Telemetry{
			{
				LapId:      "lap-3",
				Speed:      61.5,
				Gear:       4,
				TrackName:  "Silverstone",
				Throttle:   0.75,
				LapDistPct: 0.25,
				TickTime:   timestamppb.New(fixedTick),
			},
			{
				LapId:      "lap-3",
				Speed:      58.25,
				Gear:       3,
				TrackName:  "Silverstone",
				Throttle:   0.5,
				LapDistPct: 0.5,
				TickTime:   timestamppb.New(fixedTick.Add(time.Second)),
			},
		},
	}
}

func bigBatch(t *testing.T, n int) *messaging.TelemetryBatch {
	t.Helper()

	batch := &messaging.TelemetryBatch{
		BatchId:   "batch-big",
		SessionId: "sess-big",
		Records:   make([]*messaging.Telemetry, n),
	}
	for i := range batch.Records {
		batch.Records[i] = &messaging.Telemetry{
			LapId:              "lap-1",
			Speed:              40 + float64(i%60),
			Rpm:                4000 + float64(i%3000),
			Gear:               uint32(i%6 + 1),
			TrackName:          "Silverstone Grand Prix Circuit",
			SessionType:        "Race",
			Lat:                52.0786 + float64(i%1000)*1e-6,
			Lon:                -1.0169 + float64(i%1000)*1e-6,
			SteeringWheelAngle: float64(i%90) / 90.0,
			TickTime:           timestamppb.New(fixedTick.Add(time.Duration(i) * time.Millisecond)),
		}
	}
	return batch
}

func TestHandleIngest_ZstdProtobuf_Accepted(t *testing.T) {
	fake := &fakeWriter{}
	s := newTestServer(fake)

	batch := sampleBatch()
	rec := doIngest(s, zstdEncodeAll(t, marshal(t, batch)), "application/x-protobuf", "zstd")

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d (body: %q)", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	calls := fake.calls()
	if len(calls) != 1 {
		t.Fatalf("writer received %d batches, want 1", len(calls))
	}
	got := calls[0]
	if len(got) != 2 {
		t.Fatalf("batch has %d records, want 2", len(got))
	}

	if got[0].SessionID != "sess-42" {
		t.Errorf("SessionID = %q, want %q", got[0].SessionID, "sess-42")
	}
	if got[0].CarID != "car-7" {
		t.Errorf("CarID = %q, want %q", got[0].CarID, "car-7")
	}

	if got[0].Speed != 61.5 {
		t.Errorf("Speed = %v, want 61.5", got[0].Speed)
	}
	if got[0].LapID != "lap-3" {
		t.Errorf("LapID = %q, want %q", got[0].LapID, "lap-3")
	}
	if got[0].Gear != 4 {
		t.Errorf("Gear = %d, want 4", got[0].Gear)
	}
	if !got[0].Timestamp.Equal(fixedTick) {
		t.Errorf("Timestamp = %v, want %v", got[0].Timestamp, fixedTick)
	}
	if got[1].Speed != 58.25 {
		t.Errorf("record 1 Speed = %v, want 58.25", got[1].Speed)
	}

	if got[0] == got[1] {
		t.Error("records alias the same pointer")
	}
}

func TestHandleIngest_UncompressedProtobuf_Accepted(t *testing.T) {
	fake := &fakeWriter{}
	s := newTestServer(fake)

	rec := doIngest(s, marshal(t, sampleBatch()), "application/x-protobuf", "")

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	calls := fake.calls()
	if len(calls) != 1 || len(calls[0]) != 2 {
		t.Fatalf("writer got %d batches, want 1 of 2 records", len(calls))
	}
	if calls[0][0].SessionID != "sess-42" {
		t.Errorf("SessionID = %q, want %q", calls[0][0].SessionID, "sess-42")
	}
}

func TestHandleIngest_CorruptZstdBody_BadRequest(t *testing.T) {
	fake := &fakeWriter{}
	s := newTestServer(fake)

	rec := doIngest(s, []byte("this is definitely not a zstd frame"), "application/x-protobuf", "zstd")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if n := len(fake.calls()); n != 0 {
		t.Errorf("writer received %d batches, want 0", n)
	}
}

func TestHandleIngest_ValidZstdNonProtobuf_BadRequest(t *testing.T) {
	fake := &fakeWriter{}
	s := newTestServer(fake)

	rec := doIngest(s, zstdEncodeAll(t, bytes.Repeat([]byte{0xFF}, 64)), "application/x-protobuf", "zstd")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if n := len(fake.calls()); n != 0 {
		t.Errorf("writer received %d batches, want 0", n)
	}
}

func TestHandleIngest_Bomb_DeclaredSize_PayloadTooLarge(t *testing.T) {
	fake := &fakeWriter{}
	s := newTestServer(fake)

	payload := zstdEncodeAll(t, make([]byte, maxIngestBytes*4))

	var hdr zstd.Header
	if err := hdr.Decode(payload); err != nil {
		t.Fatalf("decode header: %v", err)
	}
	if !hdr.HasFCS {
		t.Fatal("expected a frame WITH FrameContentSize; this test would otherwise duplicate the unknown-size case")
	}

	rec := doIngest(s, payload, "application/x-protobuf", "zstd")

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	if n := len(fake.calls()); n != 0 {
		t.Errorf("writer received %d batches, want 0", n)
	}
}

func TestHandleIngest_Bomb_UnknownSize_PayloadTooLarge(t *testing.T) {
	fake := &fakeWriter{}
	s := newTestServer(fake)

	payload := zstdEncodeStream(t, make([]byte, maxIngestBytes*4))

	var hdr zstd.Header
	if err := hdr.Decode(payload); err != nil {
		t.Fatalf("decode header: %v", err)
	}
	if hdr.HasFCS {
		t.Fatal("expected a frame WITHOUT FrameContentSize")
	}

	rec := doIngest(s, payload, "application/x-protobuf", "zstd")

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	if n := len(fake.calls()); n != 0 {
		t.Errorf("writer received %d batches, want 0", n)
	}
}

func TestDecodeZstd_BoundedAllocation(t *testing.T) {
	const cap = 1 << 20 // 1 MiB

	dec, err := zstd.NewReader(nil, zstd.WithDecoderMaxMemory(cap))
	if err != nil {
		t.Fatalf("zstd.NewReader: %v", err)
	}
	defer dec.Close()

	bomb := zstdEncodeStream(t, make([]byte, 256*cap)) // 256 MiB of expansion

	out, err := dec.DecodeAll(bomb, nil)
	if !errors.Is(err, zstd.ErrDecoderSizeExceeded) && !errors.Is(err, zstd.ErrWindowSizeExceeded) {
		t.Fatalf("err = %v, want ErrDecoderSizeExceeded or ErrWindowSizeExceeded", err)
	}
	if len(out) != 0 {
		t.Errorf("returned %d bytes, want 0", len(out))
	}

	res := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = dec.DecodeAll(bomb, nil)
		}
	})
	if got := res.AllocedBytesPerOp(); got > 4*cap {
		t.Fatalf("decoding a 256x bomb allocated %d bytes/op, want <= %d", got, 4*cap)
	}
}

func TestHandleIngest_OversizedCompressedBody_PayloadTooLarge(t *testing.T) {
	fake := &fakeWriter{}
	s := newTestServer(fake)

	body := make([]byte, maxIngestBytes+1)
	if _, err := crand.Read(body); err != nil {
		t.Fatalf("rand: %v", err)
	}

	rec := doIngest(s, body, "application/x-protobuf", "zstd")

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestHandleIngest_OversizedUncompressedBody_PayloadTooLarge(t *testing.T) {
	fake := &fakeWriter{}
	s := newTestServer(fake)

	body := make([]byte, maxIngestBytes+1)
	if _, err := crand.Read(body); err != nil {
		t.Fatalf("rand: %v", err)
	}

	rec := doIngest(s, body, "application/x-protobuf", "")

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestHandleIngest_ContentType(t *testing.T) {
	body := marshal(t, sampleBatch())

	tests := []struct {
		contentType string
		want        int
	}{
		{"application/x-protobuf", http.StatusAccepted},
		{"application/x-protobuf; charset=utf-8", http.StatusAccepted},
		{"Application/X-Protobuf", http.StatusAccepted},
		{"application/json", http.StatusUnsupportedMediaType},
		{"", http.StatusUnsupportedMediaType},
		{"not/a/valid/type", http.StatusUnsupportedMediaType},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			s := newTestServer(&fakeWriter{})
			rec := doIngest(s, body, tt.contentType, "")
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestHandleIngest_ContentEncoding(t *testing.T) {
	raw := marshal(t, sampleBatch())

	gzipped := func() []byte {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		_, _ = w.Write(raw)
		_ = w.Close()
		return buf.Bytes()
	}()

	tests := []struct {
		name     string
		encoding string
		body     []byte
		want     int
	}{
		{"absent means uncompressed", "", raw, http.StatusAccepted},
		{"zstd", "zstd", zstdEncodeAll(t, raw), http.StatusAccepted},
		{"case insensitive", "ZSTD", zstdEncodeAll(t, raw), http.StatusAccepted},
		{"gzip unsupported", "gzip", gzipped, http.StatusUnsupportedMediaType},
		{"brotli unsupported", "br", raw, http.StatusUnsupportedMediaType},
		{"encoding chain unsupported", "zstd, gzip", zstdEncodeAll(t, raw), http.StatusUnsupportedMediaType},
		{"identity unsupported", "identity", raw, http.StatusUnsupportedMediaType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestServer(&fakeWriter{})
			rec := doIngest(s, tt.body, "application/x-protobuf", tt.encoding)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestHandleIngest_QueueFull_TooManyRequests(t *testing.T) {
	fake := &fakeWriter{err: domain.ErrQueueFull}
	s := newTestServer(fake)

	rec := doIngest(s, marshal(t, sampleBatch()), "application/x-protobuf", "")

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
}

func TestHandleIngest_WriterError_InternalServerError(t *testing.T) {
	fake := &fakeWriter{err: errors.New("questdb unreachable")}
	s := newTestServer(fake)

	rec := doIngest(s, marshal(t, sampleBatch()), "application/x-protobuf", "")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandleIngest_EmptyBody_AcceptedWithNoRecords(t *testing.T) {
	fake := &fakeWriter{}
	s := newTestServer(fake)

	rec := doIngest(s, nil, "application/x-protobuf", "")

	// Documents current behaviour: an empty batch is still enqueued.
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	calls := fake.calls()
	if len(calls) != 1 {
		t.Fatalf("writer received %d batches, want 1", len(calls))
	}
	if len(calls[0]) != 0 {
		t.Errorf("batch has %d records, want 0", len(calls[0]))
	}
}

// TestHandleIngest_ProductionSizedBatch_Accepted fails if maxIngestBytes drops below a real batch.
func TestHandleIngest_ProductionSizedBatch_Accepted(t *testing.T) {
	fake := &fakeWriter{}
	s := newTestServer(fake)

	const records = 24000 // BATCH_SIZE_RECORDS
	raw := marshal(t, bigBatch(t, records))
	compressed := zstdEncodeAll(t, raw)

	t.Logf("production-sized batch: raw %.2f MiB, zstd %.2f MiB (%.2fx), cap %.0f MiB",
		float64(len(raw))/(1<<20), float64(len(compressed))/(1<<20),
		float64(len(raw))/float64(len(compressed)), float64(maxIngestBytes)/(1<<20))

	if len(raw) >= maxIngestBytes {
		t.Fatalf("uncompressed production batch is %d bytes, which exceeds maxIngestBytes (%d)", len(raw), maxIngestBytes)
	}
	if len(compressed) >= maxIngestBytes {
		t.Fatalf("compressed production batch is %d bytes, which exceeds maxIngestBytes (%d)", len(compressed), maxIngestBytes)
	}

	rec := doIngest(s, compressed, "application/x-protobuf", "zstd")

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	calls := fake.calls()
	if len(calls) != 1 || len(calls[0]) != records {
		t.Fatalf("writer got %d batches, want 1 of %d records", len(calls), records)
	}
}

// TestHandleIngest_ThroughRouter_ZstdAccepted is the only test through chi, proving
// the response compressor and request decompression don't interfere.
func TestHandleIngest_ThroughRouter_ZstdAccepted(t *testing.T) {
	fake := &fakeWriter{}
	s := NewServer(":0", nil, fake)

	req := httptest.NewRequest(http.MethodPost, "/api/ingest",
		bytes.NewReader(zstdEncodeAll(t, marshal(t, sampleBatch()))))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "zstd")
	req.Header.Set("Accept-Encoding", "br, gzip")

	rec := httptest.NewRecorder()
	s.app.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d (body: %q)", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	if n := len(fake.calls()); n != 1 {
		t.Errorf("writer received %d batches, want 1", n)
	}
}
