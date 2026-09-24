package testfixtures_test

import (
	"bytes"
	"testing"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/messaging"
	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/testfixtures"
	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"
)

func marshalTicks(t *testing.T, n int, seed int64) []byte {
	t.Helper()

	recs, err := messaging.TransformStructBatch(testfixtures.Ticks(n, seed))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := proto.Marshal(&messaging.TelemetryBatch{Records: recs, BatchId: "b", SessionId: "s"})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// TestTicks_Deterministic pins reproducibility. Tests that assert on exact
// sizes are only meaningful if the same seed yields the same bytes.
func TestTicks_Deterministic(t *testing.T) {
	if !bytes.Equal(marshalTicks(t, 500, 7), marshalTicks(t, 500, 7)) {
		t.Fatal("same seed produced different bytes")
	}
	if bytes.Equal(marshalTicks(t, 500, 7), marshalTicks(t, 500, 8)) {
		t.Fatal("different seeds produced identical bytes")
	}
}

// TestTicks_CompressesLikeRealTelemetry guards the fixture against being
// simplified into rand.Float64(), which compresses ~1.0x. The fixture measures
// ~3.4x (denser than the real corpus's 2.99x); the floor sits below both.
func TestTicks_CompressesLikeRealTelemetry(t *testing.T) {
	const (
		n        = 24000 // BATCH_SIZE_RECORDS
		minRatio = 2.8
	)

	raw := marshalTicks(t, n, 1)

	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest), zstd.WithEncoderConcurrency(1))
	if err != nil {
		t.Fatal(err)
	}
	defer enc.Close()
	wire := enc.EncodeAll(raw, nil)

	ratio := float64(len(raw)) / float64(len(wire))
	t.Logf("raw   %6.1f B/tick (real 315.0)", float64(len(raw))/n)
	t.Logf("zstd  %6.1f B/tick (real 105.4)", float64(len(wire))/n)
	t.Logf("ratio %6.2fx        (real 2.99x)", ratio)

	if ratio < minRatio {
		t.Fatalf("fixture compresses %.2fx, want >= %.2fx — the generator no longer "+
			"resembles real telemetry, so ratio-sensitive tests elsewhere are not "+
			"measuring anything", ratio, minRatio)
	}
}
