package processing

import (
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/messaging"
	"github.com/OJPARKINSON/ibt"
	"github.com/OJPARKINSON/ibt/headers"
	"google.golang.org/protobuf/proto"
)

// collector is a minimal ibt.Processor that keeps copies of every tick,
// mirroring what production does before it ships ticks over the bus.
type collector struct {
	fields any
	ticks  []*ibt.TelemetryTick
}

func (c *collector) Init(_ *headers.Session) error { return nil }
func (c *collector) ProcessStruct(tick *ibt.TelemetryTick, _ bool) error {
	cp := *tick // value copy — the parser reuses the pointer
	c.ticks = append(c.ticks, &cp)
	return nil
}
func (c *collector) Fields() any          { return c.fields }
func (c *collector) FlushPendingData() error { return nil }
func (c *collector) Close() error         { return nil }
func (c *collector) GetMetrics() any      { return nil }

func gz(b []byte) int {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	_, _ = w.Write(b)
	_ = w.Close()
	return buf.Len()
}

func TestWireSize_TicksVsFile(t *testing.T) {
	dir := os.Getenv("IBT_DIR")
	if dir == "" {
		dir = "../../../ibt_files"
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("no ibt dir: %v", err)
	}
	var file string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".ibt" {
			file = filepath.Join(dir, e.Name())
			break
		}
	}
	if file == "" {
		t.Skip("no .ibt file found")
	}

	rawInfo, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	rawBytes, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	stubs, err := ibt.ParseStubs(file)
	if err != nil {
		t.Fatal(err)
	}

	// borrow production's Fields() shape
	lp := &loaderProcessor{}

	var allTicks []*ibt.TelemetryTick
	for _, group := range stubs.Group() {
		c := &collector{fields: lp.Fields()}
		if err := ibt.Process(context.Background(), group, c); err != nil {
			t.Fatal(err)
		}
		allTicks = append(allTicks, c.ticks...)
	}

	protoTicks, err := messaging.TransformStructBatch(allTicks)
	if err != nil {
		t.Fatal(err)
	}

	batch := &messaging.TelemetryBatch{Records: protoTicks}
	wire, err := proto.Marshal(batch)
	if err != nil {
		t.Fatal(err)
	}

	rawSize := rawInfo.Size()
	tickSize := int64(len(wire))
	rawGz := int64(gz(rawBytes))
	tickGz := int64(gz(wire))

	mb := func(b int64) float64 { return float64(b) / (1024 * 1024) }

	t.Logf("file:            %s", filepath.Base(file))
	t.Logf("ticks:           %d", len(allTicks))
	t.Logf("")
	t.Logf("RAW ibt:         %7.2f MB", mb(rawSize))
	t.Logf("PROTO ticks:     %7.2f MB   (%.2fx raw)", mb(tickSize), float64(tickSize)/float64(rawSize))
	t.Logf("")
	t.Logf("gzip RAW ibt:    %7.2f MB", mb(rawGz))
	t.Logf("gzip PROTO ticks:%7.2f MB   (%.2fx gzip-raw)", mb(tickGz), float64(tickGz)/float64(rawGz))
	t.Logf("")
	t.Logf("bytes/tick (proto): %d", tickSize/int64(len(allTicks)))
}
