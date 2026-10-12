// Times one lap from bytes to TelemetryDataPoint[] in three forms: row JSON (today),
// column JSON and the columnar blob. Inputs come from the Go test (learnings/columnar.md):
//
//	WIRE_COLUMNAR=1 IBT_DIR=<corpus> COLUMNAR_OUT=<dir> go test ./internal/processing -run TestColumnar$
//	node dashboard/scripts/columnar-bench.mjs <dir>
//
// Needs Node >= 22.15 for zlib.zstdDecompressSync. Benchmark only; nothing imports this.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import {
	brotliCompressSync,
	constants,
	gunzipSync,
	gzipSync,
	zstdDecompressSync,
} from "node:zlib";

const dir = process.argv[2];
if (!dir) {
	console.error("usage: node columnar-bench.mjs <COLUMNAR_OUT dir>");
	process.exit(1);
}
const RUNS = 50;

// Copy of the row loop in processIRacingDataWithGPS (lib/Fetch.tsx:22-60).
function fromRows(envelope) {
	const rows = envelope.points;
	const trackName = envelope.track_name ?? "";
	const sessionNum = envelope.session_num ?? "";
	const points = [];
	for (let i = 0; i < rows.length; i++) {
		const d = rows[i];
		const lat = d.lat || 0;
		const lon = d.lon || 0;
		if (!lat || !lon || Math.abs(lat) > 90 || Math.abs(lon) > 180) continue;
		points.push({
			index: points.length,
			time: d._time || points.length,
			sessionTime: d.session_time || 0,
			Speed: d.speed ? d.speed * 3.6 : 0,
			RPM: d.rpm || 0,
			Throttle: d.throttle ? d.throttle * 100 : 0,
			Brake: d.brake ? d.brake * 100 : 0,
			Gear: d.gear || 0,
			LapDistPct: d.lap_dist_pct ? d.lap_dist_pct * 100 : 0,
			SteeringWheelAngle: d.steering_wheel_angle || 0,
			Lat: lat,
			Lon: lon,
			VelocityX: d.velocity_x || 0,
			VelocityY: d.velocity_y || 0,
			VelocityZ: d.velocity_z || 0,
			FuelLevel: d.fuel_level || 0,
			LapCurrentLapTime: d.lap_current_lap_time || 0,
			PlayerCarPosition: d.player_car_position || 0,
			TrackName: trackName,
			SessionNum: sessionNum,
			LatAccel: d.lat_accel || 0,
		});
	}
	return smooth(points);
}

// Same loop reading one array per channel (column JSON arrays or blob typed arrays).
function fromColumns(c) {
	const trackName = c.track_name ?? "";
	const sessionNum = c.session_num ?? "";
	const n = c.speed.length;
	const points = [];
	for (let i = 0; i < n; i++) {
		const lat = c.lat[i] || 0;
		const lon = c.lon[i] || 0;
		if (!lat || !lon || Math.abs(lat) > 90 || Math.abs(lon) > 180) continue;
		const speed = c.speed[i];
		const throttle = c.throttle[i];
		const brake = c.brake[i];
		const pct = c.lap_dist_pct[i];
		points.push({
			index: points.length,
			time: points.length, // the API sends no _time, so the row path lands here too
			sessionTime: c.session_time[i] || 0,
			Speed: speed ? speed * 3.6 : 0,
			RPM: c.rpm[i] || 0,
			Throttle: throttle ? throttle * 100 : 0,
			Brake: brake ? brake * 100 : 0,
			Gear: c.gear[i] || 0,
			LapDistPct: pct ? pct * 100 : 0,
			SteeringWheelAngle: c.steering_wheel_angle[i] || 0,
			Lat: lat,
			Lon: lon,
			VelocityX: c.velocity_x[i] || 0,
			VelocityY: c.velocity_y[i] || 0,
			VelocityZ: c.velocity_z[i] || 0,
			FuelLevel: c.fuel_level[i] || 0,
			LapCurrentLapTime: c.lap_current_lap_time[i] || 0,
			PlayerCarPosition: c.player_car_position[i] || 0,
			TrackName: trackName,
			SessionNum: sessionNum,
			LatAccel: c.lat_accel[i] || 0,
		});
	}
	return smooth(points);
}

// Copy of the 5-point moving average in processIRacingDataWithGPS (lib/Fetch.tsx:78-126).
function smooth(points) {
	const n = points.length;
	const origLat = new Float64Array(n);
	const origLon = new Float64Array(n);
	const origSpeed = new Float64Array(n);
	for (let i = 0; i < n; i++) {
		origLat[i] = points[i].Lat;
		origLon[i] = points[i].Lon;
		origSpeed[i] = points[i].Speed;
	}
	if (n >= 5) {
		let sumLat = 0;
		let sumLon = 0;
		let sumSpeed = 0;
		let speedCount = 0;
		for (let i = 0; i < 5; i++) {
			sumLat += origLat[i];
			sumLon += origLon[i];
			if (origSpeed[i]) {
				sumSpeed += origSpeed[i];
				speedCount++;
			}
		}
		for (let center = 2; center <= n - 3; center++) {
			points[center].Lat = sumLat / 5;
			points[center].Lon = sumLon / 5;
			points[center].Speed =
				speedCount > 0 ? sumSpeed / speedCount : origSpeed[center];
			const out = center - 2;
			const inIdx = center + 3;
			if (inIdx < n) {
				sumLat += origLat[inIdx] - origLat[out];
				sumLon += origLon[inIdx] - origLon[out];
				if (origSpeed[out]) {
					sumSpeed -= origSpeed[out];
					speedCount--;
				}
				if (origSpeed[inIdx]) {
					sumSpeed += origSpeed[inIdx];
					speedCount++;
				}
			}
		}
	}
	return points;
}

// Decodes encodeFrame (ingest/go/internal/processing/columnar_test.go) into
// { name: Float32Array | Float64Array, …, string runs as their first value }.
function decodeBlob(b) {
	const dv = new DataView(b.buffer, b.byteOffset, b.byteLength);
	const dec = new TextDecoder();
	let p = 0;
	const name = () => {
		const l = b[p++];
		p += l;
		return dec.decode(b.subarray(p - l, p));
	};
	const n = dv.getUint32(0, true);
	const xor = (b[4] & 1) === 1;
	const ncols = b[5];
	p = 6;
	const out = {};
	for (let c = 0; c < ncols; c++) {
		const nm = name();
		const w = b[p++];
		const bytes = new Uint8Array(n * w);
		for (let k = 0; k < w; k++) {
			const src = p + k * n;
			for (let i = 0; i < n; i++) bytes[i * w + k] = b[src + i];
		}
		p += n * w;
		if (xor) {
			// XOR is bitwise, so a 64-bit word is undone as its two 32-bit halves.
			const u = new Uint32Array(bytes.buffer);
			const stride = w / 4;
			for (let i = stride; i < u.length; i++) u[i] ^= u[i - stride];
		}
		out[nm] =
			w === 4 ? new Float32Array(bytes.buffer) : new Float64Array(bytes.buffer);
	}
	if (b[p++] === 1) p += n * 8; // timestamps: the API blob has none
	const nstrs = b[p++];
	for (let s = 0; s < nstrs; s++) {
		const nm = name();
		const runs = dv.getUint32(p, true);
		p += 4;
		for (let r = 0; r < runs; r++) {
			const l = dv.getUint16(p + 4, true); // skip the run's count
			p += 6;
			// The envelope keeps the first point's value, so the first run wins.
			if (r === 0) out[nm] = dec.decode(b.subarray(p, p + l));
			p += l;
		}
	}
	return out;
}

const rowsJson = readFileSync(join(dir, "lap.rows.json"));
const colsJson = readFileSync(join(dir, "lap.cols.json"));
const blobZstd = readFileSync(join(dir, "lap.blob"));
const blobRaw = zstdDecompressSync(blobZstd);
const rowsGz = gzipSync(rowsJson, { level: 5 });
const colsGz = gzipSync(colsJson, { level: 5 });
const blobGz = gzipSync(blobRaw, { level: 5 });

const forms = [
	{
		name: "row JSON (today)",
		wire: rowsGz,
		raw: rowsJson,
		unzip: gunzipSync,
		decode: (b) => fromRows(JSON.parse(b.toString())),
	},
	{
		name: "column JSON",
		wire: colsGz,
		raw: colsJson,
		unzip: gunzipSync,
		decode: (b) => fromColumns(JSON.parse(b.toString())),
	},
	{
		name: "blob, zstd",
		wire: blobZstd,
		raw: blobRaw,
		unzip: zstdDecompressSync,
		decode: (b) => fromColumns(decodeBlob(b)),
	},
	{
		name: "blob, gzip-5",
		wire: blobGz,
		raw: blobRaw,
		unzip: gunzipSync,
		decode: (b) => fromColumns(decodeBlob(b)),
	},
];

const median = (fn) => {
	for (let i = 0; i < 5; i++) fn(); // warm the JIT
	const t = [];
	for (let i = 0; i < RUNS; i++) {
		const s = performance.now();
		fn();
		t.push(performance.now() - s);
	}
	t.sort((a, b) => a - b);
	return t[t.length >> 1];
};

const want = forms[0].decode(forms[0].raw);
assert.ok(want.length > 0, "row JSON produced no points");
for (const f of forms.slice(1))
	assert.deepStrictEqual(
		f.decode(f.raw),
		want,
		`${f.name} rows differ from row JSON`,
	);

const ticks = JSON.parse(rowsJson.toString()).points.length;
console.log(
	`lap: ${ticks} ticks → ${want.length} TelemetryDataPoint rows, identical across all forms`,
);
console.log(`node ${process.version}, median of ${RUNS} runs\n`);
// The API registers brotli ahead of gzip (telemetryService server.go), so HTTPS browsers get br-5.
const br5 = (b) =>
	brotliCompressSync(b, { params: { [constants.BROTLI_PARAM_QUALITY]: 5 } })
		.length;
console.log(
	"| Form | KB raw | KB on wire | KB brotli-5 | parse + transform ms | incl. decompress ms |",
);
console.log("|---|---:|---:|---:|---:|---:|");
for (const f of forms) {
	const parse = median(() => f.decode(f.raw));
	const full = median(() => f.decode(f.unzip(f.wire)));
	const kb = (n) => (n / 1024).toFixed(1);
	console.log(
		`| ${f.name} | ${kb(f.raw.length)} | ${kb(f.wire.length)} | ${kb(br5(f.raw))} | ${parse.toFixed(2)} | ${full.toFixed(2)} |`,
	);
}
