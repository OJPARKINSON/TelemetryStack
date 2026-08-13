# Dashboard

Vite + React 19 + TanStack Router frontend for browsing iRacing telemetry sessions — synchronised track maps (MapLibre) and telemetry charts (Recharts), reading from the Go telemetry service's REST API over QuestDB.

## Stack

- **Vite** — build tool / dev server
- **React 19 + TanStack Router** — app shell and routing
- **MapLibre GL** — track map rendering and racing-line overlays
- **Recharts** — telemetry charts (speed, throttle, brake, etc.)
- **SWR** — data fetching against the telemetry service API
- **Tailwind CSS** — styling
- **Biome** — lint/format
- **Playwright** — end-to-end tests

## Development

```bash
pnpm install
pnpm dev          # starts on :3000
```

The dev server expects the telemetry service API to be reachable (see the root `README.md` for running the full stack via `make restart-dev`, or point at a running instance via env config).

## Scripts

| Command | Description |
|---|---|
| `pnpm dev` | Start dev server |
| `pnpm build` | Type-check and build for production |
| `pnpm test:build` | Type-check only (`tsc --noEmit`) |
| `pnpm test` | Run Playwright e2e tests |
| `pnpm test:ui` | Playwright UI mode |
| `pnpm lighthouse` | Run Lighthouse CI audits |
| `pnpm lint` | Biome check + write |

## Docker

```bash
docker build -t dashboard .
```

Built and served as part of the root `docker-compose.yml` / `docker-compose.dev.yml`, routed via Traefik at `/dashboard`.

## Docs

See [`docs/RACING_LINE.md`](docs/RACING_LINE.md) for design notes on the racing-line map/chart sync feature.
