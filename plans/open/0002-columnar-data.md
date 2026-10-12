---
area: ingest
opened: 2026-10-12
base: 5dda901
branch: 0002-columnar-data
blocked-on: paused 2026-10-12 (you); was waiting on step 3 QuestDB Parquet run on the M4 podman stack
---
# 0002-columnar-data

## The ask

> I want to start a new area of work scoping out moving to columular data rather than the current method. Verify this and look at the work that has gone before. Also compare against atlas data server etc

## Answered

- Which layer? > the whole pipeline including and or excluding questdb
- What to verify? > That columnar still beats what we do now
- Which systems to compare? > Any top level motorsports teams use (wec, f1)
- Who runs the QuestDB step? > it's running now on m4 14c with 24gb ram but the containers have less resources set with podman
- Is the dashboard in scope? > yeah whole pipline includes dashboard

## Overview

Re-measure, stage by stage, whether a columnar layout still beats today's row-wise pipeline, and
compare it with what F1 and WEC teams actually run. The output is a committed, re-runnable
measurement and one learnings doc with a per-stage verdict. Nothing changes on the production path.

## Current state

Every stage is row-wise today.

| Stage | Today | Where |
|---|---|---|
| ingest encode | each `ibt.TelemetryTick` → one `Telemetry` proto message, 39 numeric fields + 3 strings + timestamp per row | `ingest/go/internal/messaging/transform_struct.go:12`, `telemetry.proto:8-52` |
| wire | `TelemetryBatch{repeated Telemetry}` (~24k records), proto.Marshal → zstd `SpeedFastest` | `pubSub.go:333-353`, `compress.go:13-26` |
| service decode | zstd decode → proto → `domain.TelemetryPoint` per row | `telemetryService/golang/internal/api/ingest.go:34-61`, `internal/domain/telemetry.go:76` |
| queue | shards by UTC day, merges, sorts by timestamp | `internal/queue/*.go:64-132` |
| storage write | ILP one row at a time: 8 SYMBOL + 34 DOUBLE + 2 INT | `adapter/questdb/writer.go:20-69` |
| storage | QuestDB native columns (uncompressed, fixed width), WAL, `PARTITION BY DAY`, DEDUP(timestamp, session_id). 4 STRING columns repeat per-session constants on every row | `adapter/questdb/schema.go:22-71` |
| read | SQL by session + lap, returns rows | `internal/db/queries/telemetry.sql:1-10` |
| API | JSON `{points: [{session_time, speed, …}, …]}`, 17 keys repeated per point, gzip level 5 | `internal/api/handlers.go:56-125`, `server.go:26` |
| dashboard | `telemetryFetcher` → `res.json()` → `processIRacingDataWithGPS` loops the rows, filters bad lat/lon, rescales units into `TelemetryDataPoint` rows; Recharts and MapLibre take rows | `dashboard/lib/Fetch.tsx:22-60,217-225`, `hooks/useTelemetryData.ts` |

Earlier work (all measurements, nothing shipped):

- `learnings/wire-format.md` (2026-09-24, 38 files, 3.65M ticks): row proto+zstd **2.99x**;
  columns narrowed to float32 + byte shuffle + zstd **5.68x**; adding XOR **5.91x**; Gorilla 4.55x.
  Columnar encodes 2.5–3x faster. The gain needs one session per batch in time order (+18% cost
  if interleaved, +90% if shuffled). Verdict at the time: "not worth building yet, wire bytes
  aren't the bottleneck (QuestDB O3 is)". **Its test files were not committed** (§Method describes them).
- `learnings/ordered-ingest.md` §3.2: picked zstd. `TestWireCodecs` in
  `ingest/go/internal/processing/codec_test.go:80` regenerates that table (opt-in `WIRE_CODECS=1`,
  corpus from `$IBT_DIR`, `codec_test.go:234`).
- `learnings/hosting-sizing.md:13-31,436-449`: QuestDB row ~400 B/tick (~86 MB/session-hour);
  slim schema → ~170 B; per-lap column blobs ~7–10 MB/session-hour; QuestDB
  `CONVERT PARTITION TO PARQUET` estimated 4–8x. Estimates, not measured.
- Prior research: an existing lap-analysis tool serves each lap as one columnar blob (TDF, raw LE
  arrays per channel, no timestamp column, gzip on the wire).
- `learnings/general.md`, `learnings/multicast.md`, `docs/architecture-reference.md:56-66`: team
  tooling (ATLAS, Pi Toolbox, MoTeC, WinTAX), kdb+, Kafka, tiered stores, Parquet cold archive.

Industry facts found in round one (to verify and cite in step 4):

- **ATLAS / Motion Applied.** Storage: SQL Race (SQL Server + filestream), SQLite, SSN2 (single
  file, "basic file read/write", for sharing/archive), and Parquet export described as "columnar,
  for big-data analytics / long-term storage" ([comparison](https://atlas.motionapplied.com/key-functionality/store/sqlrace/comparison/),
  [ecosystem](https://atlas.motionapplied.com/introduction/ecosystem-guide/)). Streaming: Open
  Streaming Architecture, protobuf over Kafka via a gRPC Stream API
  ([OSA](https://atlas.motionapplied.com/key-functionality/stream/open_streaming_architecture/),
  [Stream API](https://atlas.motionapplied.com/developer-resources/secu4/stream_api/)). A
  third-party project says `PeriodicDataPacket` carries per-parameter `SampleColumn`s, i.e.
  columnar; **unverified against the published .proto**.
- **kdb+** (columnar, in-memory + on-disk): Alpine F1 team-wide (KX press release); Red Bull
  partnership (older, unconfirmed current). Nothing found for Mercedes.
- **MoTeC .ld**: per-channel metadata blocks pointing at contiguous sample arrays, i.e. columnar
  (`gotzl/ldparser`, `motec-i2` crate).
- Pi Toolbox and Marelli WinTAX (WEC) internals: nothing found yet.

## End state

- `learnings/columnar.md` exists with one table: each pipeline stage → today's measured bytes (and
  encode/decode time where relevant) → columnar's measured bytes → verdict (build / don't / later)
  and why. Covers both "with QuestDB" and "without QuestDB (per-lap blobs)" storage, through to the dashboard.
- The dashboard stage has measured parse + transform time for one real lap: row JSON (today) vs
  column JSON vs columnar blob, each ending in the same `TelemetryDataPoint[]`.
- The columnar wire numbers from `wire-format.md` are re-measured on today's corpus and code, and
  the doc says whether 5.68x / 5.91x still hold (±5%).
- An industry section compares ATLAS, kdb+, MoTeC, Pi Toolbox, WinTAX and a web lap-analysis tool on wire format,
  storage layout and read path, every claim with a source link or marked unverified.
- One opt-in test (`WIRE_COLUMNAR=1`) is committed so the numbers can be regenerated.
- If any stage gets a "build" verdict, a `/plan-add` draft exists for it. This plan builds nothing.

## Not doing

- Changing `telemetry.proto`, any wire format, the QuestDB schema, the API response or the dashboard's
  production code. The dashboard gets a benchmark script only.
- Swapping Recharts/MapLibre for column-native charting.
- Implementing XOR/shuffle in production code.
- Lossy encodings (dropping `SessionTime`, quantising lat/lon) — `wire-format.md` §2 lists them as next.
- Re-litigating ordering (`ordered-ingest*.md`) or the zstd choice.
- Live streaming (iRacing memory-mapped feed).
- Swapping QuestDB for another database. "Without QuestDB" here means per-lap blobs only.

## One-way doors

None. Measurement only; no wire, schema or response type changes. A follow-up plan owns those.

## Steps

1. **Re-measure the wire (commit the lost test).** In `ingest/go/internal/processing/`, add
   `columnar_test.go`, opt-in `WIRE_COLUMNAR=1`, reusing `corpus()` and `chunker` from
   `codec_test.go`, exactly as `wire-format.md` §Method describes: row proto+zstd baseline,
   columns float32 + shuffle + zstd, + XOR. Each decoded and compared bit-for-bit
   (`math.Float64bits`), so the encoder lives in the test file only. Report MB, B/tick, ratio,
   encode MB/s, and decode MB/s (not measured before; the server pays it).
2. **Storage and API payload, offline.** Same test file, same corpus, grouped per lap
   (`tick.LapID`) instead of per batch:
   - per-lap columnar blob size (step 1 encoder, one lap per blob) → MB per session-hour;
   - QuestDB native bytes per row computed from `schema.go:22-71` column widths, current and slim
     (`hosting-sizing.md:27-29`);
   - API payload per lap: today's `telemetryEnvelope` row JSON (copy the 17 fields from
     `handlers.go:56-80`) vs column JSON (`{"speed":[…], …}`) vs the blob, each after gzip level 5
     (matches `server.go:26`).
   - When `COLUMNAR_OUT=<dir>` is set, write one median-length lap in all three forms
     (`lap.rows.json`, `lap.cols.json`, `lap.blob`) for step 4.
3. **QuestDB on your local stack (your stop point).** You run it on the M4 (14 cores, 24 GB) under
   podman. The doc records the podman machine's CPU/RAM and the QuestDB container limits (compose
   asks for 6 CPU / 16 GB, `docker-compose.yml:82-83`), since the containers get less than the host.
   The agent writes the exact commands into the doc and does not start containers:
   - partition size on disk before and after
     `ALTER TABLE TelemetryTicks CONVERT PARTITION TO PARQUET LIST '<day>'` for one ingested day;
   - `GetLapTelemetry` latency (`curl --compressed -w '%{time_total}'` on
     `/api/sessions/{id}/laps/{lap}`, 20 runs, median) before and after the convert;
   - response bytes on the wire for the same lap (`-w '%{size_download}'`), to check step 2's gzip estimate.
4. **Dashboard parse + transform.** Add `dashboard/scripts/columnar-bench.mjs` (plain Node, no new
   dependency), reading the three files from step 2. For each, time decode → `TelemetryDataPoint[]`
   over 50 runs (median): row JSON goes through a copy of `processIRacingDataWithGPS`
   (`lib/Fetch.tsx:22-60`); column JSON is parsed then zipped into the same rows; the blob is
   un-zstd'd, un-shuffled into `Float32Array`s, then zipped. Assert all three produce identical rows.
   Manual cross-check in Chrome: load one real lap and read `telemetryFetcher`'s time in the
   Performance panel, so the Node numbers are anchored to the browser.
   Note: the blob path needs zstd in the browser. If that means a dependency, record it as the
   cost and measure the blob with gzip (`DecompressionStream`, built in) instead.
5. **Industry comparison.** Verify every round-one fact above from primary sources (Motion Applied
   docs and published .proto, KX, MoTeC parsers). Search Pi Toolbox and WinTAX formats. Fill the
   comparison table: wire format, storage layout, analysis read path, columnar yes/no, source.
6. **Write `learnings/columnar.md`** in the house style of `wire-format.md` (TL;DR bullets, tables,
   Method). Link it from `wire-format.md` and `hosting-sizing.md`. Per-stage verdict, ingest to
   dashboard.

## Verification

- `cd ingest/go && go vet ./... && go test -race ./internal/...` (the new test skips without the env var).
- `cd dashboard && pnpm test:build && pnpm exec biome check`.
- `WIRE_COLUMNAR=1 IBT_DIR=<corpus> COLUMNAR_OUT=<dir> go test ./internal/processing -run TestColumnar -v -timeout 30m`
  prints every table in the doc; numbers in the doc match the output.
- `node dashboard/scripts/columnar-bench.mjs <dir>` prints the dashboard table and passes its
  identical-rows assertion.
- Manual: the step 1 baseline row matches `TestWireCodecs`' zstd-fastest row (same bytes).
- Manual: step 3 run by you on the M4 podman stack; numbers and container limits pasted in.
- Manual: Chrome Performance panel time for one real lap is in the same range as the Node row-JSON number.
- Manual: every industry claim has a link or says "unverified".

## Risks

- Corpus may have changed since 2026-09-24 (the epoch fix re-ingest); compare B/tick, not totals.
- Lap grouping: lap 0 / out-laps and offline `SubSessionID 0` sessions can skew per-lap sizes;
  report median and total.
- Step 3 runs under podman limits on a laptop, not the Pi; quote it as relative (before/after),
  not as absolute Pi latency.
- Motion Applied's protobuf schema may need an account to view; then mark the ATLAS wire claim unverified.

## Open questions

None.
