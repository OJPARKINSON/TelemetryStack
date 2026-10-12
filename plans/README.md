# plans

## Next

- [0002-columnar-data](next/0002-columnar-data.md) — measure columnar vs row-wise across the whole pipeline (ingest → QuestDB/blobs → API → dashboard) and compare with ATLAS, kdb+, MoTeC

## Open

## Blocked on

## Drafts

- [0001-dash-sector-zoom](drafts/0001-dash-sector-zoom.md): Zoom on a sector, plus a per-track geometry table


## Gates

| Area | Gate |
|---|---|
| telemetry service | `cd telemetryService/golang && go vet ./... && go test -race ./...` |
| ingest | `cd ingest/go && go vet ./... && go test -race ./internal/...` |
| dashboard | `cd dashboard && pnpm test:build && pnpm exec biome check` |

## Stop points (yours, never the loop's)

- e2e: `cd e2e && go test ./tests/ -v -short` (Docker/testcontainers), at review only
- `make restart-dev` / any docker compose (`down -v` wipes the QuestDB volume)
- `sqlc generate` in `telemetryService/golang/internal/db/` after editing `queries/*.sql`
- `ingest/go/ibt` submodule: never edited or committed
