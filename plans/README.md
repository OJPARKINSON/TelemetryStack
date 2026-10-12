# plans

## Next

- [0002-columnar-data](next/0002-columnar-data.md) — measure columnar vs row-wise across the whole pipeline (ingest → QuestDB/blobs → API → dashboard) and compare with ATLAS, kdb+, MoTeC

## Open

- [0001-dash-sector-zoom](open/0001-dash-sector-zoom.md): Zoom on a sector, plus a per-track sectors table · PR #31

## Blocked on

## Drafts



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
