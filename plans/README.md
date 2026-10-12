# plans

## Next

## Open

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
- `sqlc generate` in `telemetryService/golang/internal/db/` after editing `queries/*.sql`
- `ingest/go/ibt` submodule: never edited or committed
