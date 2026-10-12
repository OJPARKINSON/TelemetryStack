package processing

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/messaging"
	"github.com/OJPARKINSON/ibt"
	"github.com/OJPARKINSON/ibt/headers"
	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"
)

// TestColumnar regenerates the tables in learnings/columnar.md: the wire re-measure
// (wire-format.md §2) and the per-lap storage / API payload sizes. Opt-in:
//
//	WIRE_COLUMNAR=1 IBT_DIR=<corpus> COLUMNAR_OUT=<dir> go test ./internal/processing -run TestColumnar -v -timeout 30m
//
// The encoder lives here only; nothing on the production path uses it.

// colChannel is one numeric channel. The first 17 are the API's telemetryPointWire
// (telemetryService handlers.go), in its field order, under its JSON names.
type colChannel struct {
	name string
	get  func(*ibt.TelemetryTick) float64
}

var colChannels = []colChannel{
	{"session_time", func(t *ibt.TelemetryTick) float64 { return t.SessionTime }},
	{"speed", func(t *ibt.TelemetryTick) float64 { return t.Speed }},
	{"rpm", func(t *ibt.TelemetryTick) float64 { return t.RPM }},
	{"throttle", func(t *ibt.TelemetryTick) float64 { return t.Throttle }},
	{"brake", func(t *ibt.TelemetryTick) float64 { return t.Brake }},
	{"gear", func(t *ibt.TelemetryTick) float64 { return float64(t.Gear) }},
	{"lap_dist_pct", func(t *ibt.TelemetryTick) float64 { return t.LapDistPct }},
	{"steering_wheel_angle", func(t *ibt.TelemetryTick) float64 { return t.SteeringWheelAngle }},
	{"player_car_position", func(t *ibt.TelemetryTick) float64 { return t.PlayerCarPosition }},
	{"lat", func(t *ibt.TelemetryTick) float64 { return t.Lat }},
	{"lon", func(t *ibt.TelemetryTick) float64 { return t.Lon }},
	{"velocity_x", func(t *ibt.TelemetryTick) float64 { return t.VelocityX }},
	{"velocity_y", func(t *ibt.TelemetryTick) float64 { return t.VelocityY }},
	{"velocity_z", func(t *ibt.TelemetryTick) float64 { return t.VelocityZ }},
	{"lat_accel", func(t *ibt.TelemetryTick) float64 { return t.LatAccel }},
	{"lap_current_lap_time", func(t *ibt.TelemetryTick) float64 { return t.LapCurrentLapTime }},
	{"fuel_level", func(t *ibt.TelemetryTick) float64 { return t.FuelLevel }},

	{"lap_id", func(t *ibt.TelemetryTick) float64 { return float64(t.LapID) }},
	{"session_num", func(t *ibt.TelemetryTick) float64 { return float64(t.SessionNum) }},
	{"track_id", func(t *ibt.TelemetryTick) float64 { return float64(t.TrackID) }},
	{"alt", func(t *ibt.TelemetryTick) float64 { return t.Alt }},
	{"long_accel", func(t *ibt.TelemetryTick) float64 { return t.LongAccel }},
	{"vert_accel", func(t *ibt.TelemetryTick) float64 { return t.VertAccel }},
	{"pitch", func(t *ibt.TelemetryTick) float64 { return t.Pitch }},
	{"roll", func(t *ibt.TelemetryTick) float64 { return t.Roll }},
	{"yaw", func(t *ibt.TelemetryTick) float64 { return t.Yaw }},
	{"yaw_north", func(t *ibt.TelemetryTick) float64 { return t.YawNorth }},
	{"voltage", func(t *ibt.TelemetryTick) float64 { return t.Voltage }},
	{"lap_last_lap_time", func(t *ibt.TelemetryTick) float64 { return t.LapLastLapTime }},
	{"water_temp", func(t *ibt.TelemetryTick) float64 { return t.WaterTemp }},
	{"lap_delta_to_best_lap", func(t *ibt.TelemetryTick) float64 { return t.LapDeltaToBestLap }},
	{"lf_pressure", func(t *ibt.TelemetryTick) float64 { return t.LFpressure }},
	{"rf_pressure", func(t *ibt.TelemetryTick) float64 { return t.RFpressure }},
	{"lr_pressure", func(t *ibt.TelemetryTick) float64 { return t.LRpressure }},
	{"rr_pressure", func(t *ibt.TelemetryTick) float64 { return t.RRpressure }},
	{"lf_temp_m", func(t *ibt.TelemetryTick) float64 { return t.LFtempM }},
	{"rf_temp_m", func(t *ibt.TelemetryTick) float64 { return t.RFtempM }},
	{"lr_temp_m", func(t *ibt.TelemetryTick) float64 { return t.LRtempM }},
	{"rr_temp_m", func(t *ibt.TelemetryTick) float64 { return t.RRtempM }},
}

const apiChannels = 17

type colString struct {
	name string
	get  func(*ibt.TelemetryTick) string
}

var wireStrings = []colString{
	{"session_type", func(t *ibt.TelemetryTick) string { return t.SessionType }},
	{"session_name", func(t *ibt.TelemetryTick) string { return t.SessionName }},
	{"track_name", func(t *ibt.TelemetryTick) string { return t.TrackName }},
}

// The API sends these two once per lap in the envelope; the blob carries them as runs.
var apiStrings = []colString{
	{"track_name", func(t *ibt.TelemetryTick) string { return t.TrackName }},
	{"session_num", func(t *ibt.TelemetryTick) string { return strconv.Itoa(int(t.SessionNum)) }},
}

// colFrame is a column-major batch. times is nil when the frame carries no timestamps.
type colFrame struct {
	n        int
	names    []string
	cols     [][]float64
	times    []int64
	strNames []string
	strs     [][]string
}

func buildFrame(ticks []*ibt.TelemetryTick, chans []colChannel, strs []colString, withTimes bool) *colFrame {
	f := &colFrame{n: len(ticks)}
	for _, c := range chans {
		col := make([]float64, len(ticks))
		for i, t := range ticks {
			col[i] = c.get(t)
		}
		f.names = append(f.names, c.name)
		f.cols = append(f.cols, col)
	}
	if withTimes {
		f.times = make([]int64, len(ticks))
		for i, t := range ticks {
			f.times[i] = t.TickTime.UnixNano()
		}
	}
	for _, s := range strs {
		col := make([]string, len(ticks))
		for i, t := range ticks {
			col[i] = s.get(t)
		}
		f.strNames = append(f.strNames, s.name)
		f.strs = append(f.strs, col)
	}
	return f
}

// Frame layout, little endian (dashboard/scripts/columnar-bench.mjs decodes the same bytes):
//
//	u32 n | u8 flags (bit0 = XOR) | u8 ncols
//	ncols × { u8 nameLen, name, u8 width (4|8), n×width bytes byte-shuffled }
//	u8 hasTimes | [n×8 bytes shuffled: first unix-nano value, then deltas]
//	u8 nstrs × { u8 nameLen, name, u32 runs, runs × { u32 count, u16 len, bytes } }
//
// A column is float32 only when every value round-trips bit-for-bit, so narrowing is lossless.
func encodeFrame(f *colFrame, xor bool) []byte {
	out := binary.LittleEndian.AppendUint32(nil, uint32(f.n))
	flags := byte(0)
	if xor {
		flags = 1
	}
	out = append(out, flags, byte(len(f.cols)))

	words := make([]uint64, f.n)
	for ci, col := range f.cols {
		out = appendName(out, f.names[ci])
		w := 4
		for _, v := range col {
			if math.Float64bits(float64(float32(v))) != math.Float64bits(v) {
				w = 8
				break
			}
		}
		for i, v := range col {
			if w == 4 {
				words[i] = uint64(math.Float32bits(float32(v)))
			} else {
				words[i] = math.Float64bits(v)
			}
		}
		out = append(out, byte(w))
		out = appendShuffled(out, words, w, xor)
	}

	if f.times == nil {
		out = append(out, 0)
	} else {
		out = append(out, 1)
		prev := int64(0)
		for i, t := range f.times {
			words[i] = uint64(t - prev)
			prev = t
		}
		out = appendShuffled(out, words, 8, false)
	}

	out = append(out, byte(len(f.strs)))
	for si, col := range f.strs {
		out = appendName(out, f.strNames[si])
		var runs []int
		for i := range col {
			if i == 0 || col[i] != col[i-1] {
				runs = append(runs, i)
			}
		}
		out = binary.LittleEndian.AppendUint32(out, uint32(len(runs)))
		for ri, start := range runs {
			end := f.n
			if ri+1 < len(runs) {
				end = runs[ri+1]
			}
			out = binary.LittleEndian.AppendUint32(out, uint32(end-start))
			out = binary.LittleEndian.AppendUint16(out, uint16(len(col[start])))
			out = append(out, col[start]...)
		}
	}
	return out
}

func appendName(out []byte, name string) []byte {
	return append(append(out, byte(len(name))), name...)
}

// appendShuffled writes byte 0 of every value, then byte 1, … (Parquet BYTE_STREAM_SPLIT).
// XOR is done in place, back to front, so words holds the XOR'd values afterwards.
func appendShuffled(out []byte, words []uint64, w int, xor bool) []byte {
	if xor {
		for i := len(words) - 1; i > 0; i-- {
			words[i] ^= words[i-1]
		}
	}
	for k := 0; k < w; k++ {
		shift := 8 * k
		for _, v := range words {
			out = append(out, byte(v>>shift))
		}
	}
	return out
}

func decodeFrame(b []byte) (*colFrame, error) {
	p := 0
	need := func(k int) error {
		if p+k > len(b) {
			return fmt.Errorf("frame truncated at %d (+%d of %d)", p, k, len(b))
		}
		return nil
	}
	u8 := func() byte { p++; return b[p-1] }
	name := func() string { l := int(u8()); p += l; return string(b[p-l : p]) }
	unshuffle := func(n, w int, xor bool) []uint64 {
		words := make([]uint64, n)
		for k := 0; k < w; k++ {
			shift := 8 * k
			for i := range words {
				words[i] |= uint64(b[p+k*n+i]) << shift
			}
		}
		p += n * w
		if xor {
			for i := 1; i < n; i++ {
				words[i] ^= words[i-1]
			}
		}
		return words
	}

	if err := need(6); err != nil {
		return nil, err
	}
	f := &colFrame{n: int(binary.LittleEndian.Uint32(b))}
	p = 4
	xor := u8()&1 == 1
	ncols := int(u8())
	for range ncols {
		if err := need(1); err != nil {
			return nil, err
		}
		nm := name()
		w := int(u8())
		if err := need(f.n * w); err != nil {
			return nil, err
		}
		words := unshuffle(f.n, w, xor)
		col := make([]float64, f.n)
		for i, v := range words {
			if w == 4 {
				col[i] = float64(math.Float32frombits(uint32(v)))
			} else {
				col[i] = math.Float64frombits(v)
			}
		}
		f.names = append(f.names, nm)
		f.cols = append(f.cols, col)
	}

	if u8() == 1 {
		if err := need(f.n * 8); err != nil {
			return nil, err
		}
		words := unshuffle(f.n, 8, false)
		f.times = make([]int64, f.n)
		prev := int64(0)
		for i, d := range words {
			prev += int64(d)
			f.times[i] = prev
		}
	}

	nstrs := int(u8())
	for range nstrs {
		nm := name()
		runs := int(binary.LittleEndian.Uint32(b[p:]))
		p += 4
		col := make([]string, 0, f.n)
		for range runs {
			count := int(binary.LittleEndian.Uint32(b[p:]))
			l := int(binary.LittleEndian.Uint16(b[p+4:]))
			p += 6
			s := string(b[p : p+l])
			p += l
			for range count {
				col = append(col, s)
			}
		}
		f.strNames = append(f.strNames, nm)
		f.strs = append(f.strs, col)
	}
	if p != len(b) {
		return nil, fmt.Errorf("frame has %d trailing bytes", len(b)-p)
	}
	return f, nil
}

// sameFrame compares bit-for-bit (math.Float64bits), so NaN payloads count.
func sameFrame(a, b *colFrame) error {
	if a.n != b.n || !slices.Equal(a.names, b.names) || !slices.Equal(a.strNames, b.strNames) {
		return fmt.Errorf("shape differs: n %d/%d", a.n, b.n)
	}
	for c := range a.cols {
		for i := range a.cols[c] {
			if math.Float64bits(a.cols[c][i]) != math.Float64bits(b.cols[c][i]) {
				return fmt.Errorf("%s[%d]: %v != %v", a.names[c], i, a.cols[c][i], b.cols[c][i])
			}
		}
	}
	if !slices.Equal(a.times, b.times) {
		return fmt.Errorf("times differ")
	}
	for s := range a.strs {
		if !slices.Equal(a.strs[s], b.strs[s]) {
			return fmt.Errorf("strings %s differ", a.strNames[s])
		}
	}
	return nil
}

func TestColumnarFrameRoundTrip(t *testing.T) {
	base := time.Date(2026, 9, 24, 19, 0, 0, 0, time.UTC)
	var ticks []*ibt.TelemetryTick
	for i := range 100 {
		tk := &ibt.TelemetryTick{
			Speed:       float64(float32(40 + 0.1*float64(i))),
			Lat:         51.5 + 1e-7*float64(i), // needs float64
			Gear:        uint32(i % 6),
			SessionTime: 1234.5 + float64(i)/60,
			TrackName:   "spa",
			SessionType: "Practice",
			TickTime:    base.Add(time.Duration(i) * time.Second / 60),
		}
		if i == 50 {
			tk.TrackName = "spa 2024"
			tk.RPM = math.NaN()
		}
		ticks = append(ticks, tk)
	}
	for _, xor := range []bool{false, true} {
		for _, withTimes := range []bool{false, true} {
			want := buildFrame(ticks, colChannels, wireStrings, withTimes)
			got, err := decodeFrame(encodeFrame(want, xor))
			if err != nil {
				t.Fatal(err)
			}
			if err := sameFrame(want, got); err != nil {
				t.Fatalf("xor=%v times=%v: %v", xor, withTimes, err)
			}
		}
	}
}

// Copies of telemetryService/golang/internal/api/handlers.go telemetryPointWire / telemetryEnvelope.
type colPointWire struct {
	SessionTime        float64 `json:"session_time"`
	Speed              float64 `json:"speed"`
	RPM                float64 `json:"rpm"`
	Throttle           float64 `json:"throttle"`
	Brake              float64 `json:"brake"`
	Gear               uint32  `json:"gear"`
	LapDistPct         float64 `json:"lap_dist_pct"`
	SteeringWheelAngle float64 `json:"steering_wheel_angle"`
	PlayerCarPosition  float64 `json:"player_car_position"`
	Lat                float64 `json:"lat"`
	Lon                float64 `json:"lon"`
	VelocityX          float64 `json:"velocity_x"`
	VelocityY          float64 `json:"velocity_y"`
	VelocityZ          float64 `json:"velocity_z"`
	LatAccel           float64 `json:"lat_accel"`
	LapCurrentLapTime  float64 `json:"lap_current_lap_time"`
	FuelLevel          float64 `json:"fuel_level"`
}

type colEnvelope struct {
	TrackName  string         `json:"track_name"`
	SessionNum string         `json:"session_num"`
	Points     []colPointWire `json:"points"`
}

func lapRowsJSON(ticks []*ibt.TelemetryTick) ([]byte, error) {
	env := colEnvelope{
		TrackName:  ticks[0].TrackName,
		SessionNum: strconv.Itoa(int(ticks[0].SessionNum)),
		Points:     make([]colPointWire, len(ticks)),
	}
	for i, p := range ticks {
		env.Points[i] = colPointWire{p.SessionTime, p.Speed, p.RPM, p.Throttle, p.Brake, p.Gear,
			p.LapDistPct, p.SteeringWheelAngle, p.PlayerCarPosition, p.Lat, p.Lon,
			p.VelocityX, p.VelocityY, p.VelocityZ, p.LatAccel, p.LapCurrentLapTime, p.FuelLevel}
	}
	return json.Marshal(env)
}

// lapColsJSON is the same envelope with one array per channel: {"track_name":…,"speed":[…],…}.
func lapColsJSON(f *colFrame, ticks []*ibt.TelemetryTick) ([]byte, error) {
	var buf bytes.Buffer
	head, _ := json.Marshal(map[string]string{
		"track_name":  ticks[0].TrackName,
		"session_num": strconv.Itoa(int(ticks[0].SessionNum)),
	})
	buf.Write(head[:len(head)-1])
	for c, col := range f.cols {
		b, err := json.Marshal(col)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.names[c], err)
		}
		fmt.Fprintf(&buf, ",%q:", f.names[c])
		buf.Write(b)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// gzip5 matches the API's middleware.NewCompressor(5) (telemetryService server.go).
func gzip5(b []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, 5)
	_, _ = w.Write(b)
	_ = w.Close()
	return buf.Bytes()
}

// lapper hands ticks off one lap at a time: a run of equal LapID within one stub group.
// Unlike chunker it stamps the session metadata the way loaderProcessor.ProcessStruct does,
// so the string columns (QuestDB STRING bytes, API track_name) carry production values.
type lapper struct {
	fields  any
	session *headers.Session
	buf     []*ibt.TelemetryTick
	flush   func([]*ibt.TelemetryTick) error
}

func (l *lapper) Init(s *headers.Session) error { l.session = s; return nil }

func (l *lapper) ProcessStruct(tick *ibt.TelemetryTick, _ bool) error {
	if l.session != nil {
		tick.TrackName = l.session.WeekendInfo.TrackDisplayShortName
		tick.TrackID = l.session.WeekendInfo.TrackID
		for _, s := range l.session.SessionInfo.Sessions {
			if s.SessionNum == int(tick.SessionNum) {
				tick.SessionType, tick.SessionName = s.SessionType, s.SessionName
			}
		}
	}
	if len(l.buf) > 0 && l.buf[len(l.buf)-1].LapID != tick.LapID {
		if err := l.FlushPendingData(); err != nil {
			return err
		}
	}
	cp := *tick // value copy — the parser reuses the pointer
	l.buf = append(l.buf, &cp)
	return nil
}

func (l *lapper) FlushPendingData() error {
	if len(l.buf) == 0 {
		return nil
	}
	err := l.flush(l.buf)
	l.buf = nil // flush may keep the slice
	return err
}

func (l *lapper) Fields() any     { return l.fields }
func (l *lapper) Close() error    { return l.FlushPendingData() }
func (l *lapper) GetMetrics() any { return nil }

// walkCorpus runs proc over every stub group of every file, in sorted order.
func walkCorpus(t *testing.T, files []string, newProc func() ibt.Processor) {
	t.Helper()
	for _, file := range files {
		stubs, err := ibt.ParseStubs(file)
		if err != nil {
			t.Errorf("%s: parse stubs: %v", filepath.Base(file), err)
			continue
		}
		groups := stubs.Group()
		for _, group := range groups {
			p := newProc()
			if err := ibt.Process(context.Background(), group, p); err != nil {
				t.Fatalf("%s: process: %v", filepath.Base(file), err)
			}
			if err := p.Close(); err != nil {
				t.Fatalf("%s: flush tail: %v", filepath.Base(file), err)
			}
		}
		ibt.CloseAllStubs(groups)
	}
}

type colStat struct {
	name     string
	bytes    int64
	enc, dec time.Duration
}

func TestColumnar(t *testing.T) {
	if os.Getenv("WIRE_COLUMNAR") == "" {
		t.Skip("set WIRE_COLUMNAR=1 to run the columnar corpus measurement")
	}
	files := corpus(t)
	fields := (&loaderProcessor{}).Fields() // production's channel whitelist

	zenc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest), zstd.WithEncoderConcurrency(1))
	if err != nil {
		t.Fatal(err)
	}
	defer zenc.Close()
	zdec, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
	if err != nil {
		t.Fatal(err)
	}
	defer zdec.Close()

	mb := func(b int64) float64 { return float64(b) / (1024 * 1024) }

	// ---- Step 1: wire, production-shaped batches ----
	var (
		rawProto   int64
		wireTicks  int64
		wireBatchN int
	)
	protoStat := &colStat{name: "proto + zstd-fastest (production)"}
	shufStat := &colStat{name: "columns, float32 + shuffle + zstd"}
	xorStat := &colStat{name: "columns, float32 + XOR + shuffle + zstd"}

	measureWire := func(ticks []*ibt.TelemetryTick) error {
		wireTicks += int64(len(ticks))
		wireBatchN++

		// Same bytes as TestWireCodecs: TransformStructBatch → Marshal → zstd.
		start := time.Now()
		records, err := messaging.TransformStructBatch(ticks)
		if err != nil {
			return err
		}
		wire, err := proto.Marshal(&messaging.TelemetryBatch{Records: records})
		if err != nil {
			return err
		}
		z := zenc.EncodeAll(wire, nil)
		protoStat.enc += time.Since(start)
		rawProto += int64(len(wire))
		protoStat.bytes += int64(len(z))

		start = time.Now()
		plain, err := zdec.DecodeAll(z, nil)
		if err != nil {
			return err
		}
		var back messaging.TelemetryBatch
		if err := proto.Unmarshal(plain, &back); err != nil {
			return err
		}
		protoStat.dec += time.Since(start)
		if len(back.Records) != len(ticks) {
			return fmt.Errorf("proto decoded %d of %d records", len(back.Records), len(ticks))
		}

		for _, s := range []struct {
			st  *colStat
			xor bool
		}{{shufStat, false}, {xorStat, true}} {
			start := time.Now()
			f := buildFrame(ticks, colChannels, wireStrings, true)
			z := zenc.EncodeAll(encodeFrame(f, s.xor), nil)
			s.st.enc += time.Since(start)
			s.st.bytes += int64(len(z))

			start = time.Now()
			plain, err := zdec.DecodeAll(z, nil)
			if err != nil {
				return err
			}
			got, err := decodeFrame(plain)
			s.st.dec += time.Since(start)
			if err != nil {
				return fmt.Errorf("%s: %w", s.st.name, err)
			}
			if err := sameFrame(f, got); err != nil {
				return fmt.Errorf("%s: %w", s.st.name, err)
			}
		}
		return nil
	}

	walkCorpus(t, files, func() ibt.Processor { return &chunker{fields: fields, flush: measureWire} })
	if wireTicks == 0 {
		t.Fatal("corpus produced no ticks")
	}

	t.Logf("")
	t.Logf("== Step 1: wire (%d ticks, %d batches of %d) ==", wireTicks, wireBatchN, batchRecords)
	t.Logf("raw proto: %.1f MB (%.1f B/tick); MB/s are raw-proto MB per second, one thread", mb(rawProto), float64(rawProto)/float64(wireTicks))
	t.Logf("| Encoding | MB | B/tick | Ratio | Encode MB/s | Decode MB/s |")
	t.Logf("|---|---:|---:|---:|---:|---:|")
	for _, s := range []*colStat{protoStat, shufStat, xorStat} {
		t.Logf("| %s | %.1f | %.1f | %.2fx | %.0f | %.0f |", s.name, mb(s.bytes),
			float64(s.bytes)/float64(wireTicks), float64(rawProto)/float64(s.bytes),
			mb(rawProto)/s.enc.Seconds(), mb(rawProto)/s.dec.Seconds())
	}

	// ---- Step 2: storage and API payload, one lap at a time ----
	type lapSizes struct {
		ticks                        int
		blob                         int // storage blob: all channels + times + strings, XOR+shuffle+zstd
		rows, rowsGz, cols, colsGz   int
		apiBlob, apiBlobZ, apiBlobGz int
	}
	var (
		laps        []lapSizes
		jsonSkipped int
		strLen      = map[string]int64{}
		lapTicks    int64
	)

	measureLap := func(ticks []*ibt.TelemetryTick) error {
		lapTicks += int64(len(ticks))
		for _, tk := range ticks {
			strLen["track_name"] += int64(len(tk.TrackName))
			strLen["session_type"] += int64(len(tk.SessionType))
			strLen["track_id"] += int64(len(strconv.Itoa(int(tk.TrackID))))
		}

		ls := lapSizes{ticks: len(ticks)}
		ls.blob = len(zenc.EncodeAll(encodeFrame(buildFrame(ticks, colChannels, wireStrings, true), true), nil))

		rows, err := lapRowsJSON(ticks)
		api := buildFrame(ticks, colChannels[:apiChannels], apiStrings, false)
		cols, err2 := lapColsJSON(api, ticks)
		if err != nil || err2 != nil {
			jsonSkipped++ // NaN/Inf: encoding/json refuses them, so the API would 500 on this lap
		} else {
			ls.rows, ls.rowsGz = len(rows), len(gzip5(rows))
			ls.cols, ls.colsGz = len(cols), len(gzip5(cols))
		}
		raw := encodeFrame(api, true)
		ls.apiBlob, ls.apiBlobZ, ls.apiBlobGz = len(raw), len(zenc.EncodeAll(raw, nil)), len(gzip5(raw))
		laps = append(laps, ls)
		return nil
	}

	walkCorpus(t, files, func() ibt.Processor { return &lapper{fields: fields, flush: measureLap} })

	hours := float64(lapTicks) / (60 * 3600) // shortcut: assumes 60 Hz .ibt; upgrade if a 360 Hz corpus appears
	sum := func(get func(lapSizes) int) (total int64, median int) {
		vals := make([]int, 0, len(laps))
		for _, l := range laps {
			if v := get(l); v > 0 {
				total += int64(v)
				vals = append(vals, v)
			}
		}
		slices.Sort(vals)
		if len(vals) > 0 {
			median = vals[len(vals)/2]
		}
		return total, median
	}

	tickCounts := make([]int, len(laps))
	for i, l := range laps {
		tickCounts[i] = l.ticks
	}
	slices.Sort(tickCounts)
	medianTicks := tickCounts[len(tickCounts)/2]

	t.Logf("")
	t.Logf("== Step 2: per lap (%d laps, %d ticks, %.2f session-hours at 60 Hz, median lap %d ticks) ==",
		len(laps), lapTicks, hours, medianTicks)

	blobTotal, blobMedian := sum(func(l lapSizes) int { return l.blob })
	protoPerTick := float64(protoStat.bytes) / float64(wireTicks)
	t.Logf("| Storage | B/tick | MB per session-hour | Median lap KB |")
	t.Logf("|---|---:|---:|---:|")

	// QuestDB native bytes per row from schema.go:22-71. SYMBOL = 4 B key; STRING = 8 B .i offset
	// + 4 B length + UTF-16 chars; INDEX (session_id, lap_id) = 8 B row id each in .v.
	// car_id is never set by ingest, so it is the empty string.
	avg := func(k string) float64 { return float64(strLen[k]) / float64(lapTicks) }
	str := func(chars float64) float64 { return 8 + 4 + 2*chars }
	current := 4*4.0 + str(avg("track_name")) + str(avg("session_type")) + str(0) + str(avg("track_id")) +
		2*4 + 34*8 + 8 + 2*8
	// Slim (hosting-sizing.md): the 4 STRINGs become SYMBOLs, FLOAT except lat/lon/session_time.
	slim := 8*4.0 + 2*4 + 3*8 + 31*4 + 8 + 2*8
	perHour := func(bPerTick float64) float64 { return bPerTick * 60 * 3600 / (1024 * 1024) }
	t.Logf("| QuestDB native, current schema | %.1f | %.1f | — |", current, perHour(current))
	t.Logf("| QuestDB native, slim schema | %.1f | %.1f | — |", slim, perHour(slim))
	t.Logf("| proto + zstd (wire, for scale) | %.1f | %.1f | — |", protoPerTick, perHour(protoPerTick))
	t.Logf("| per-lap blob, XOR + shuffle + zstd | %.1f | %.1f | %.1f |",
		float64(blobTotal)/float64(lapTicks), mb(blobTotal)/hours, float64(blobMedian)/1024)
	t.Logf("strings, mean chars/row: track_name %.1f, session_type %.1f, track_id %.1f",
		avg("track_name"), avg("session_type"), avg("track_id"))

	t.Logf("")
	t.Logf("API payload per lap (17 API channels; %d laps skipped for NaN/Inf in JSON)", jsonSkipped)
	t.Logf("| Form | total MB raw | total MB gzip-5 | median lap KB raw | median lap KB on wire | B/tick on wire |")
	t.Logf("|---|---:|---:|---:|---:|---:|")
	jsonTicks := int64(0)
	for _, l := range laps {
		if l.rows > 0 {
			jsonTicks += int64(l.ticks)
		}
	}
	for _, r := range []struct {
		name     string
		raw, wir func(lapSizes) int
		ticks    int64
	}{
		{"row JSON (today)", func(l lapSizes) int { return l.rows }, func(l lapSizes) int { return l.rowsGz }, jsonTicks},
		{"column JSON", func(l lapSizes) int { return l.cols }, func(l lapSizes) int { return l.colsGz }, jsonTicks},
		{"blob, zstd", func(l lapSizes) int { return l.apiBlob }, func(l lapSizes) int { return l.apiBlobZ }, lapTicks},
		{"blob, gzip-5", func(l lapSizes) int { return l.apiBlob }, func(l lapSizes) int { return l.apiBlobGz }, lapTicks},
	} {
		rt, rm := sum(r.raw)
		wt, wm := sum(r.wir)
		t.Logf("| %s | %.1f | %.1f | %.1f | %.1f | %.1f |", r.name, mb(rt), mb(wt),
			float64(rm)/1024, float64(wm)/1024, float64(wt)/float64(r.ticks))
	}

	// ---- COLUMNAR_OUT: one median-length lap in all three forms ----
	out := os.Getenv("COLUMNAR_OUT")
	if out == "" {
		return
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	var picked []*ibt.TelemetryTick
	pick := func(ticks []*ibt.TelemetryTick) error {
		if picked == nil && len(ticks) == medianTicks && ticks[0].LapID > 0 {
			if _, err := lapRowsJSON(ticks); err == nil {
				picked = ticks
			}
		}
		return nil
	}
	walkCorpus(t, files, func() ibt.Processor { return &lapper{fields: fields, flush: pick} })
	if picked == nil {
		t.Fatalf("no lap with %d ticks to write", medianTicks)
	}
	rows, _ := lapRowsJSON(picked)
	api := buildFrame(picked, colChannels[:apiChannels], apiStrings, false)
	cols, err := lapColsJSON(api, picked)
	if err != nil {
		t.Fatal(err)
	}
	blob := zenc.EncodeAll(encodeFrame(api, true), nil)
	for name, b := range map[string][]byte{"lap.rows.json": rows, "lap.cols.json": cols, "lap.blob": blob} {
		if err := os.WriteFile(filepath.Join(out, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("wrote lap %d of %s (%d ticks) to %s", picked[0].LapID, picked[0].TrackName, len(picked), out)
}
