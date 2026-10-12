# plans

## Next


## Open

<<<<<<< HEAD
- [0001-dash-sector-zoom](open/0001-dash-sector-zoom.md): Zoom on a sector, plus a per-track sectors table · PR #31
=======
- [0002-columnar-data](open/0002-columnar-data.md) — measure columnar vs row-wise across the whole pipeline (ingest → QuestDB/blobs → API → dashboard) and compare with ATLAS, kdb+, MoTeC
>>>>>>> bb7eb24 (plans: dispatch 0002)

## Blocked on

<<<<<<< HEAD
- 0001-dash-sector-zoom: you (review findings in the plan, manual stack checks, merge PR #31)

## Drafts

=======
- 0002-columnar-data: BLOCKED on you running step 3 (QuestDB Parquet on the M4 podman stack); commands in learnings/columnar.md

## Drafts

- [0001-dash-sector-zoom](drafts/0001-dash-sector-zoom.md): Zoom on a sector, plus a per-track geometry table
- [0003-columnar-lap-api](drafts/0003-columnar-lap-api.md): serve a lap as a columnar blob and decode it in the dashboard (from 0002)
>>>>>>> a61ed0a (columnar: opt-in wire/storage/api measurement test, dashboard bench, 0003 draft)


## Gates

| Area | Gate |
|---|---|
| telemetry service | `cd telemetryService/golang && go vet ./... && go test -race ./...` |
| ingest | `cd ingest/go && go vet ./... && go test -race ./internal/...` |
| dashboard | `cd dashboard && pnpm test:build && pnpm exec biome check` |

## Stop points (yours, never the loop's)

- e2e: `cd e2e && go test ./tests/ -v -short` (Docker/testcontainers), at review only
- `make restart-dev` / any docker compose (`down -v` wipes the QuestDB volume)
- `ingest/go/ibt` submodule: never edited or committed
