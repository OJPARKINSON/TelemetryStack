---
area: api
ticket: 0003
opened: 2026-10-12
---
# 0003-columnar-lap-api

## The ask
> From 0002-columnar-data (learnings/columnar.md §1): the API + dashboard stage got a "build" verdict.
> Serve a lap as a columnar blob instead of row JSON, and decode it in the dashboard.

## Related work
- `learnings/columnar.md` §3 and §5: median lap 725 KB gzip'd row JSON vs 209 KB blob (3.5x);
  dashboard decompress + parse + transform 12.3 ms vs 1.3 ms, identical `TelemetryDataPoint[]`.
- Encoder/decoder already written as measurement code: `ingest/go/internal/processing/columnar_test.go`
  (`encodeFrame`, `decodeFrame`) and `dashboard/scripts/columnar-bench.mjs` (`decodeBlob`, `fromColumns`).
- Endpoint: `telemetryService/golang/internal/api/handlers.go:56-125` (`telemetryEnvelope`),
  compressor at `server.go:26-29` (brotli registered ahead of gzip; only listed content types are compressed).
- Dashboard: `telemetryFetcher` and `processIRacingDataWithGPS` in `dashboard/lib/Fetch.tsx:22-126,217-225`.
- `player_car_position` is wrong end to end (int read as float): `learnings/columnar.md` §5 side finding.

## Open questions
- New endpoint (or `Accept` negotiation) alongside the JSON one, or replace it?
- XOR or not: 3% fewer bytes vs simpler decoder.
- float32 narrowing is lossless only against the ibt values; QuestDB stores DOUBLE, so does the server
  narrow per column the same way, and is that still bit-exact for what QuestDB returns?
- Content type for the blob, and adding it to the chi compressor's list.
- Does the geojson endpoint (`/laps/{lap}/geojson`) move too, or stay as is?
- Fix `player_car_position` in the same change or separately?
