// Package testfixtures generates synthetic telemetry that compresses like real
// iRacing data (~315 B/tick, ~3x zstd). It is a normal package, not _test.go,
// so both the messaging and processing tests can share it.
//
// The ratio only holds if the generator keeps three properties of real data:
// the same field set on every tick (field tags repeat at a fixed stride),
// realistic narrow value ranges (a float64's high bytes stay constant), and
// sensor noise in the low mantissa bytes (without it the fixture compresses 10x+).
package testfixtures

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/OJPARKINSON/ibt"
)

// Epoch is the session start used by Ticks.
var Epoch = mustDate(2025, 6, 10, 17, 25, 58)

const (
	tickHz  = 60.0 // iRacing sample rate
	lapSecs = 90.0 // synthetic lap length
)

// Ticks returns n synthetic telemetry ticks at 60 Hz, deterministic for a given
// seed. Every field that TransformStructBatch puts on the wire is populated, so
// the marshalled size per tick lands near production's ~315 bytes.
func Ticks(n int, seed int64) []*ibt.TelemetryTick {
	r := rand.New(rand.NewPCG(uint64(seed), 0x9E3779B97F4A7C15))

	ticks := make([]*ibt.TelemetryTick, n)
	fuel := 60.0

	for i := range ticks {
		t := float64(i) / tickHz
		phase := math.Mod(t, lapSecs) / lapSecs // 0..1 around the lap
		lap := int32(t/lapSecs) + 1

		// Track curvature: three sinusoids give six corners per lap.
		curv := 0.55*math.Sin(2*math.Pi*phase) +
			0.30*math.Sin(6*math.Pi*phase+0.7) +
			0.15*math.Sin(10*math.Pi*phase+1.9)

		speed := 20 + 60*(1-math.Abs(curv))   // m/s
		accel := -55 * curv * math.Abs(curv)  // proxy for dv/dt
		lateral := curv * speed * speed / 220 // v^2/r

		// Pedals are mostly hard on or fully off, which is both realistic and
		// compression-relevant: proto3 omits zero-valued scalars entirely.
		throttle := clamp(0.5+accel/30, 0, 1)
		brake := clamp(-accel/25, 0, 1)
		if brake > 0 {
			throttle = 0
		}
		if throttle > 0.98 {
			throttle = 1
		}

		yaw := 2 * math.Pi * phase
		fuel -= 0.0009

		tk := &ibt.TelemetryTick{
			LapID:      lap,
			SessionNum: 0,
			LapDistPct: f32(phase),

			Speed:    f32(speed + noise(r, 0.05)),
			RPM:      f32(1200 + 6800*math.Mod(speed/12, 1) + noise(r, 5)),
			Gear:     uint32(clamp(1+speed/12, 1, 6)),
			Throttle: f32(throttle),
			Brake:    f32(brake),

			SteeringWheelAngle: f32(2.6*curv + noise(r, 0.002)),
			PlayerCarPosition:  f32(3),

			Lat: 52.0786 + 0.004*math.Sin(yaw) + noise(r, 4e-7),
			Lon: -1.0169 + 0.006*math.Cos(yaw) + noise(r, 6e-7),
			Alt: f32(130.0 + 3*math.Sin(2*yaw) + noise(r, 0.003)),

			VelocityX: f32(speed * math.Cos(yaw)),
			VelocityY: f32(speed * math.Sin(yaw)),
			VelocityZ: 0,

			LatAccel:  f32(lateral + noise(r, 0.01)),
			LongAccel: f32(accel/3 + noise(r, 0.01)),
			VertAccel: f32(9.81 + noise(r, 0.02)),

			Pitch:    f32(0.02*math.Sin(4*yaw) + noise(r, 2e-5)),
			Roll:     f32(0.03*curv + noise(r, 2e-5)),
			Yaw:      f32(yaw),
			YawNorth: f32(math.Mod(yaw+1.2, 2*math.Pi)),

			FuelLevel: f32(fuel),
			Voltage:   f32(13.8 + 0.05*math.Sin(yaw/3) + noise(r, 5e-5)),
			WaterTemp: f32(88 + 2*math.Sin(yaw/7) + noise(r, 0.004)),

			LapCurrentLapTime: f32(phase * lapSecs),
			LapLastLapTime:    f32(88.421),
			LapDeltaToBestLap: f32(0.4*math.Sin(3*yaw) + noise(r, 0.001)),

			// Phase-shifted so corners are similar but not identical, like real sensors.
			LFpressure: f32(138.0 + 1.5*math.Sin(yaw) + noise(r, 0.01)),
			RFpressure: f32(138.4 + 1.5*math.Sin(yaw+0.4) + noise(r, 0.01)),
			LRpressure: f32(137.6 + 1.4*math.Sin(yaw+0.8) + noise(r, 0.01)),
			RRpressure: f32(137.9 + 1.4*math.Sin(yaw+1.2) + noise(r, 0.01)),

			LFtempM: f32(82 + 6*math.Sin(yaw) + noise(r, 0.02)),
			RFtempM: f32(84 + 6*math.Sin(yaw+0.4) + noise(r, 0.02)),
			LRtempM: f32(79 + 5*math.Sin(yaw+0.8) + noise(r, 0.02)),
			RRtempM: f32(81 + 5*math.Sin(yaw+1.2) + noise(r, 0.02)),

			TrackName:   "Silverstone Grand Prix Circuit",
			TrackID:     391,
			SessionType: "Race",
			SessionName: "RACE",
		}

		tk.SessionTime = t
		tk.TickTime = Epoch.Add(secs(t))
		ticks[i] = tk
	}

	return ticks
}

// noise dirties the low mantissa bytes the way a real sensor does.
func noise(r *rand.Rand, amp float64) float64 {
	return amp * (2*r.Float64() - 1)
}

// f32 quantizes to float32 precision. iRacing stores most channels as float32,
// so real doubles carry trailing zero bytes that account for much of the ~3x ratio.
func f32(v float64) float64 {
	return float64(float32(v))
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(math.Max(v, lo), hi)
}

func secs(f float64) time.Duration {
	return time.Duration(f * float64(time.Second))
}

func mustDate(y, mo, d, h, mi, s int) time.Time {
	return time.Date(y, time.Month(mo), d, h, mi, s, 0, time.UTC)
}
