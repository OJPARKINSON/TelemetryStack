package processing

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/messaging"
	"github.com/OJPARKINSON/ibt"
	"github.com/OJPARKINSON/ibt/headers"
	kpgzip "github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"
)

// TestWireCodecs regenerates the codec table in learnings/ordered-ingest.md 3.2
// from the real corpus. Opt-in:
//
//	WIRE_CODECS=1 go test ./internal/processing -run TestWireCodecs -v

// batchRecords mirrors BATCH_SIZE_RECORDS; zstd's ratio depends on batch size.
const batchRecords = 24000

// chunker hands ticks off in batch-sized groups so peak memory is one batch, not the corpus.
type chunker struct {
	fields any
	buf    []*ibt.TelemetryTick
	flush  func([]*ibt.TelemetryTick) error
}

func (c *chunker) Init(_ *headers.Session) error { return nil }

func (c *chunker) ProcessStruct(tick *ibt.TelemetryTick, _ bool) error {
	cp := *tick // value copy — the parser reuses the pointer
	c.buf = append(c.buf, &cp)
	if len(c.buf) < batchRecords {
		return nil
	}
	return c.FlushPendingData()
}

func (c *chunker) FlushPendingData() error {
	if len(c.buf) == 0 {
		return nil
	}
	err := c.flush(c.buf)
	c.buf = c.buf[:0]
	return err
}

func (c *chunker) Fields() any     { return c.fields }
func (c *chunker) Close() error    { return c.FlushPendingData() }
func (c *chunker) GetMetrics() any { return nil }

type codecStat struct {
	name   string
	bytes  int64
	dur    time.Duration
	encode func(dst *bytes.Buffer, src []byte) error
}

func (s *codecStat) run(src []byte) error {
	var buf bytes.Buffer
	buf.Grow(len(src) / 2)

	start := time.Now()
	err := s.encode(&buf, src)
	s.dur += time.Since(start)

	s.bytes += int64(buf.Len())
	return err
}

func TestWireCodecs(t *testing.T) {
	// Opt-in: walking 3.7 GB runs well past the default timeout under -race.
	if os.Getenv("WIRE_CODECS") == "" {
		t.Skip("set WIRE_CODECS=1 to run the corpus codec measurement")
	}

	files := corpus(t)

	zenc, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedFastest),
		zstd.WithEncoderConcurrency(1),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer zenc.Close()

	// gzip at its fastest level, as a stdlib-only implementation would use.
	codecs := []*codecStat{
		{name: "none", encode: func(dst *bytes.Buffer, src []byte) error {
			_, err := dst.Write(src)
			return err
		}},
		{name: "gzip-1", encode: func(dst *bytes.Buffer, src []byte) error {
			w, err := gzip.NewWriterLevel(dst, gzip.BestSpeed)
			if err != nil {
				return err
			}
			if _, err := w.Write(src); err != nil {
				return err
			}
			return w.Close()
		}},
		// Control only. The doc's gzip-1 row (404.1 MB / 2.71x) doesn't reproduce: this
		// measures 440.12 MB / 2.49x, byte-identical to stdlib, so it isn't the library.
		{name: "gzip-1-kp", encode: func(dst *bytes.Buffer, src []byte) error {
			w, err := kpgzip.NewWriterLevel(dst, kpgzip.BestSpeed)
			if err != nil {
				return err
			}
			if _, err := w.Write(src); err != nil {
				return err
			}
			return w.Close()
		}},
		{name: "zstd-fastest", encode: func(dst *bytes.Buffer, src []byte) error {
			_, err := dst.Write(zenc.EncodeAll(src, nil))
			return err
		}},
	}

	var (
		rawIbtBytes int64
		totalTicks  int64
		batches     int
	)

	// Marshal exactly as flushBatchInternal does and feed every codec the same bytes.
	measure := func(ticks []*ibt.TelemetryTick) error {
		records, err := messaging.TransformStructBatch(ticks)
		if err != nil {
			return err
		}
		wire, err := proto.Marshal(&messaging.TelemetryBatch{Records: records})
		if err != nil {
			return err
		}

		totalTicks += int64(len(ticks))
		batches++

		for _, c := range codecs {
			if err := c.run(wire); err != nil {
				return fmt.Errorf("%s: %w", c.name, err)
			}
		}
		return nil
	}

	lp := &loaderProcessor{} // borrow production's channel whitelist

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			t.Fatal(err)
		}
		rawIbtBytes += info.Size()

		stubs, err := ibt.ParseStubs(file)
		if err != nil {
			t.Errorf("%s: parse stubs: %v", filepath.Base(file), err)
			continue
		}
		groups := stubs.Group()
		for _, group := range groups {
			c := &chunker{fields: lp.Fields(), flush: measure}
			if err := ibt.Process(context.Background(), group, c); err != nil {
				t.Fatalf("%s: process: %v", filepath.Base(file), err)
			}
			if err := c.Close(); err != nil {
				t.Fatalf("%s: flush tail: %v", filepath.Base(file), err)
			}
		}
		ibt.CloseAllStubs(groups)
	}

	if totalTicks == 0 {
		t.Fatal("corpus produced no ticks")
	}

	raw := codecs[0]
	byName := map[string]*codecStat{}
	for _, c := range codecs {
		byName[c.name] = c
	}

	mb := func(b int64) float64 { return float64(b) / (1024 * 1024) }

	// Same shape as the table in learnings/ordered-ingest.md 3.2.
	t.Logf("corpus:      %d files, %.2f MB of .ibt", len(files), mb(rawIbtBytes))
	t.Logf("ticks:       %d in %d batches (%d records/batch)", totalTicks, batches, batchRecords)
	t.Logf("proto:       %.2f MB (%.1f bytes/tick)", mb(raw.bytes), float64(raw.bytes)/float64(totalTicks))
	t.Logf("")
	t.Logf("%-14s %10s %8s %12s %12s", "codec", "MB", "ratio", "B/tick", "encode MB/s")
	for _, c := range codecs {
		thr := 0.0
		if c.dur > 0 {
			thr = mb(raw.bytes) / c.dur.Seconds()
		}
		t.Logf("%-14s %10.2f %7.2fx %12.1f %12.0f",
			c.name, mb(c.bytes), float64(raw.bytes)/float64(c.bytes),
			float64(c.bytes)/float64(totalTicks), thr)
	}

	zstdStat, gzipStat := byName["zstd-fastest"], byName["gzip-1"]

	// Measured 2.99x; asserted at 2.5x so corpus drift passes but a codec or field-set regression fails.
	if ratio := float64(raw.bytes) / float64(zstdStat.bytes); ratio < 2.5 {
		t.Errorf("zstd ratio %.2fx < 2.5x floor — compression no longer pays for itself", ratio)
	}

	// zstd has to beat gzip on size to justify the dependency.
	if zstdStat.bytes >= gzipStat.bytes {
		t.Errorf("zstd %.2f MB >= gzip %.2f MB — the stdlib codec would do",
			mb(zstdStat.bytes), mb(gzipStat.bytes))
	}
	if zstdStat.dur >= gzipStat.dur {
		t.Errorf("zstd %v >= gzip %v to encode — 3.2 claims zstd is also faster",
			zstdStat.dur, gzipStat.dur)
	}
}

// corpus returns the .ibt files in $IBT_DIR, else ../../../ibt_files (ingest/ibt_files),
// skipping with the directory it tried.
func corpus(t *testing.T) []string {
	t.Helper()

	dir := os.Getenv("IBT_DIR")
	if dir == "" {
		dir = "../../../ibt_files"
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("no corpus at %s (set IBT_DIR): %v", dir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".ibt" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	if len(files) == 0 {
		t.Skipf("no .ibt files in %s (set IBT_DIR)", dir)
	}

	// Sorted so batch boundaries, and therefore ratios, are reproducible.
	sort.Strings(files)
	t.Logf("corpus dir:  %s (%d files)", dir, len(files))
	return files
}
