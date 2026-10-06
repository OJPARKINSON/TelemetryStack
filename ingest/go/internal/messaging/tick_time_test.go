package messaging

import (
	"testing"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/testfixtures"
)

// TestAddStructRecords_TickTimeFromDiskHeader: wire timestamps anchor to the
// disk header (tick.TickTime), not the filename epoch, unless the header has none.
func TestAddStructRecords_TickTimeFromDiskHeader(t *testing.T) {
	wrongEpoch := testfixtures.Epoch.AddDate(1, 0, 0)
	ticks := testfixtures.Ticks(100, 1)
	ticks[0].TickTime = time.Unix(0, 0) // header without StartDate → filename fallback

	ps := NewPubSub("sess-1", wrongEpoch, testConfig("http://unused"), nil, 0)
	t.Cleanup(func() { ps.recordBatch = nil; _ = ps.Close() }) // nothing to flush, no client

	if err := ps.AddStructRecords(ticks); err != nil {
		t.Fatal(err)
	}

	for i, rec := range ps.recordBatch {
		want := ticks[i].TickTime
		if i == 0 {
			want = wrongEpoch.Add(time.Duration(ticks[0].SessionTime * float64(time.Second)))
		}
		if got := rec.TickTime.AsTime(); !got.Equal(want) {
			t.Fatalf("tick %d: got %v, want %v", i, got, want)
		}
	}
}
