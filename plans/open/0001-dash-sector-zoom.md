---
area: dash
ticket: 0001
opened: 2026-10-12
base: 5dda901c395ced84a31b0ff5f797f4f1fe9ca840
branch: 0001-dash-sector-zoom
pr: https://github.com/OJPARKINSON/TelemetryStack/pull/31
---
# 0001-dash-sector-zoom

## The ask
> Ticket number can be 0001 we are going to start work on implemnting our own version of zoom on a sector
>
> For this I think we need another table store that stores the track data by track id so that we don't have to save it for every tick, there can just be a check to see if the track id exists already when loading an ibt file
>
> Then we need the zoom func where it zooms into the track map for where that sector is but when deselecting it zooms out
>
> maybe there is a better way to do this?

Answers (2026-10-12):
> One track would be a row in a track table
> I dont have my own outline data uness it can be infered or we get it from else where
> This isn't at the splitting stage yet

## Overview
Ingest sends each IBT file's track ID, name and iRacing sector boundaries to a new endpoint. The endpoint stores them as one row per track in a new `Tracks` table. Clicking a sector button then fits the map to that sector's part of the lap, and clicking Full Lap fits it back to the whole lap.

The "better way" is that the outline is never stored, because the lap's own lat/lon already is the outline. The only thing that needs a track table is the sector boundaries. QuestDB's `DEDUP` replaces the "does it exist" check, so re-ingesting a file overwrites the same row instead of adding another.

## Current state
- Ticks carry `track_id` and `track_name` on every row (`telemetryService/golang/internal/adapter/questdb/schema.go:27,30`). No other table exists, and the schema DDL creates only `TelemetryTicks` (`schema.go:20-86`, called at `cmd/telemetry-service/main.go:27`).
- Ingest already parses iRacing's sector boundaries but never sends them: `SplitTimeInfo.Sectors[].SectorStartPct` (fractions 0–1, the first one is 0) at `ingest/go/ibt/headers/session.go:69,238-246`. Track ID and name come from `WeekendInfo` (`ingest/go/internal/processing/processor.go:92-93`). Per file group, `groupHeaders.SessionInfo` is a `*headers.Session` available at `internal/processing/file.go:118`, before `ibt.Process` runs at `file.go:142`.
- Ingest posts only to `cfg.IngestURL` (default `SERVER_URL:8010/api/ingest`, `internal/config/config.go:64`) using the shared `fp.client` (`internal/messaging/pubSub.go:301`).
- Service routes are at `internal/api/server.go:55-62`. The repository interface is `domain.SessionRepository` (`internal/domain/*.go:10-14`), implemented by `adapter/questdb/reader.go` on top of sqlc queries (`internal/db/queries/*.sql`, sqlc schema `internal/db/schema/create.sql`).
- In the dashboard, `SectorBar` hard-codes boundaries `[0, 33.33, 66.66, 100]` (`dashboard/components/efecto/SectorBar.tsx:51`) on `LapDistPct` (0–100). `activeSector` is set in `TelemetrySection.tsx:56,142-146` but nothing reads it. The map is created with a fixed `center` = the first point and `zoom={15}` (`TelemetrySection.tsx:109-115`), and nothing fits it to the track.
- Map children reach the MapLibre instance through `useMap()` inside a `useEffect` with try/catch; `components/telemetry/RacingLine.tsx:10-52` is the pattern.
- The session page fetches the lap's geojson with SWR at `dashboard/src/routes/$sessionId.tsx:115`.

## End state
- After ingesting an IBT, `SELECT * FROM Tracks` in the QuestDB console has exactly one row for that `track_id`, holding iRacing's sector boundaries. Ingesting the same file again still leaves one row.
- `curl --compressed localhost:8010/api/sessions/{id}/laps/{lap}/track` returns that row, and returns 404 for a lap whose track was never sent.
- On the session page there is one sector button per iRacing sector (5 at Daytona), with the boundaries from the API. Clicking S2 animates the map to fit S2's part of the lap. Clicking Full Lap, or clicking the active sector again, animates it back to the whole lap. On load the map fits the whole lap.
- With no track row (data ingested before this change), the sector buttons are hidden and Full Lap fit still works.

## Not doing
- Storing or inferring a track outline or track edges (the lap's lat/lon is the outline).
- Zooming the charts' x axis, drag-to-zoom, zoom in the URL, and the range strip (`learnings/sectors-and-zoom.md` steps 1, 3, 4, 6).
- Interpolated sector times or best/optimal sectors. `SectorBar` keeps its nearest-sample times, now on the real boundaries.
- Sector tick marks, labels or start/finish on the map.
- Merging iRacing's splits into 3 "classic" sectors.
- Fixing the SubSessionID 0 collision (see Risks).
- Backfilling `Tracks` for data already ingested. Re-ingesting the files fills it, and that is idempotent.

## One-way doors

**QuestDB table** (in `adapter/questdb/schema.go`, run next to the `TelemetryTicks` DDL):
```sql
CREATE TABLE IF NOT EXISTS Tracks (
    track_id SYMBOL,
    track_name STRING,
    sector_start_pct STRING,   -- JSON array of fractions 0..1, ascending, first = 0, e.g. "[0,0.137196,0.273519,0.353413,0.630555]"
    ts TIMESTAMP               -- always 1970-01-01: with the DEDUP key this makes one row per track_id
) TIMESTAMP(ts) PARTITION BY YEAR
WAL
DEDUP UPSERT KEYS(ts, track_id);
```
Cardinality: one row per `track_id`, and the latest write wins. `track_id` and `sector_start_pct` are never null; the handler rejects them otherwise. `STRING` rather than `DOUBLE[]` so sqlc and the PG wire need no array support.

**sqlc** (`internal/db/schema/create.sql` gets a matching Postgres-typed `Tracks` with `track_id TEXT, track_name TEXT, sector_start_pct TEXT, ts TIMESTAMP`). New file `internal/db/queries/tracks.sql`; regenerate with `cd telemetryService/golang/internal/db && sqlc generate`:
```sql
-- name: UpsertTrack :exec
INSERT INTO Tracks (track_id, track_name, sector_start_pct, ts)
VALUES ($1, $2, $3, '1970-01-01T00:00:00.000000Z');

-- name: GetTrack :one
SELECT track_id, track_name, sector_start_pct FROM Tracks
WHERE track_id = $1;

-- name: GetLapTrackID :one
SELECT track_id FROM TelemetryTicks
WHERE session_id = $1 AND lap_id = $2
LIMIT 1;
```

**REST: write** (ingest → service). Returns 204, or 400 when the body is invalid:
```
PUT /api/tracks/{trackId}
Content-Type: application/json
{"track_name": "Daytona Road", "sector_start_pct": [0, 0.137196, 0.273519, 0.353413, 0.630555]}
```
Validation: `trackId` is non-empty digits; 1–64 boundaries; each in `[0,1)`; strictly ascending; the first is 0.

**REST: read** (dashboard). Returns 200, or 404 when the lap or track is unknown:
```go
type Track struct {
    TrackID        string    `json:"track_id"`
    TrackName      string    `json:"track_name"`
    SectorStartPct []float64 `json:"sector_start_pct"` // fractions 0..1
}
// GET /api/sessions/{sessionId}/laps/{lapId}/track → Track
```
```ts
// dashboard/lib/types.ts
export interface TrackInfo { track_id: string; track_name: string; sector_start_pct: number[] }
```

## Steps
1. **Service storage.** Add the `Tracks` DDL to `schema.go`, run in the same retry loop as the ticks DDL. Add the sqlc schema and `queries/tracks.sql` as above, then run `sqlc generate` in `telemetryService/golang/internal/db/` yourself (decided 2026-10-12). Add `UpsertTrack`, `GetTrack` and `GetLapTrackID` to `adapter/questdb/reader.go`, in the shape of `GetLapTelemetry` (`reader.go:68`), including `observeQuery`. `GetTrack` unmarshals the JSON string into `[]float64`. Map `pgx.ErrNoRows` to a `domain.ErrNotFound`.
2. **Service API.** Add `domain.Track` and the three methods to `SessionRepository`. Add `handlePutTrack` (validation above) and `handleGetLapTrack` to `handlers.go` in the style of `handleGetLaps` (`handlers.go:32`), and add both routes in `server.go:55`. Put a `fakeSessions` in `fakes_test.go` and write table tests for the PUT validation cases plus GET 200/404.
3. **Ingest.** New `internal/messaging/track.go` with `PutTrack(ctx, client, cfg, sess *headers.Session) error`. Its URL is `strings.TrimSuffix(cfg.IngestURL, "ingest") + "tracks/" + id`. It skips the call when `TrackID == 0` or there are no sectors. Call it in `file.go` right before `ibt.Process` (`file.go:142`) and only log a failure, because a missing track row must not fail the telemetry ingest. Test it with an `httptest` server like `helpers_test.go`: check the body, the path, and the skip cases.
4. **Dashboard.** Add `TrackInfo` to `lib/types.ts`. In `$sessionId.tsx`, SWR-fetch `/api/sessions/${sessionId}/laps/${currentLapId}/track` next to the geojson (`:115`) and pass `sector_start_pct` down through `TelemetrySection`. `SectorBar` takes `sectorStartPct?: number[]` instead of hard-coding (×100 to match `LapDistPct`, and the last sector ends at 100). It renders no sector buttons without it and cycles `SECTOR_COLORS` for more than 3 sectors. Clicking the active sector deselects it. New `components/telemetry/SectorZoom.tsx`, built like `RacingLine.tsx`: given the lap points and an optional `[startPct, endPct]`, it builds a `LngLatBounds` from the points in range (all points when null) and calls `map.fitBounds(bounds, { padding: 40, duration: 200 })`. Render it inside `MapUI` in `TelemetrySection.tsx`.

## Verification
Gates:
```bash
cd telemetryService/golang && go vet ./... && go test -race ./...
cd ingest/go && go vet ./... && go test -race ./internal/...
cd dashboard && pnpm test:build && pnpm exec biome check
```
Manual (yours; needs the stack, not `restart-dev`):
- [ ] `make restart-lite`, then ingest one Daytona IBT and check `SELECT * FROM Tracks` in the QuestDB console: 1 row with 5 boundaries. Ingest it again: still 1 row.
- [ ] `curl --compressed localhost:8010/api/sessions/{id}/laps/{lap}/track` returns 200 with the boundaries; an unknown lap returns 404.
- [ ] In the browser, the map fits the whole lap on load. S2 animates to S2, clicking S2 again or Full Lap animates back, and switching laps keeps working.
- [ ] A session ingested before this change shows no sector buttons and no console errors.
- [ ] e2e at review: `cd e2e && go test ./tests/ -v -short`.

## Risks
- **SubSessionID 0 collision.** Offline sessions share `session_id` "0", so `GetLapTrackID ... LIMIT 1` can return another track's ID for that session. This is an existing quirk; the result is wrong sector boundaries on those sessions, not a crash.
- **WAL lag.** QuestDB applies WAL writes asynchronously, so the GET can 404 for a moment right after ingest.
- **Fixed-timestamp DEDUP trick.** If a later change gives `ts` a real value, the one-row-per-track guarantee is gone. The DDL comment says so.
- `fitBounds` runs during a style transition: wrap it in try/catch like `RacingLine.tsx:39`.

## Open questions
none
