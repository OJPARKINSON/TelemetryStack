# TelemetryStack

A self-hosted telemetry pipeline for iRacing. Parses `.ibt` files, stores time-series data in QuestDB, and serves an interactive dashboard with synchronised track maps and telemetry charts — built to run on a Raspberry Pi 5.

Inspired by how professional motorsport teams process data: direct ingestion, time-series storage, and analysis tooling — no unnecessary infrastructure.

## Architecture

```
┌──────────────┐   protobuf/gRPC   ┌───────────────────┐         ┌──────────┐
│  Go Ingest   │ ────────────────▶ │ Telemetry Service │────────▶│ QuestDB  │
│  CLI (PC)    │                   │   (Go, Pi5)       │  ILP    │  (Pi5)   │
└──────────────┘                   └──────────┬────────┘         └──────────┘
                                              │ REST API
                                              │
                                        ┌─────▼─────┐
                                        │ Dashboard │
                                        │  (React)  │
                                        └───────────┘
```

**Why no message queue?** Professional teams (F1, WEC, IMSA) use direct ingestion at the garage level — sensor to logger to analysis tool. Queues only appear at factory scale when 50+ engineers need independent consumer offsets across the same data. With one producer and one consumer doing batch post-session processing, the source `.ibt` files on disk are already the durable buffer.

## What It Does

- **Parses** iRacing `.ibt` binary telemetry (mmap zero-copy, 46+ channels at 60 Hz)
- **Stores** time-series data in QuestDB (speed, throttle, brake, RPM, GPS, tire temps, G-forces, fuel, and more)
- **Visualises** laps on an interactive dashboard with synchronised track maps and telemetry charts
- **Runs** on a Pi 5 behind Traefik with Tailscale access, Prometheus metrics, and Grafana dashboards

## Stack

| Component | Tech | Role |
|---|---|---|
| **Ingest CLI** | Go | Reads `.ibt` files, mmap zero-copy parsing, batches protobuf |
| **Telemetry Service** | Go | Receives data, writes to QuestDB via ILP, serves REST API |
| **Database** | QuestDB | Time-series storage optimised for high-throughput telemetry |
| **Dashboard** | Vite + React + TanStack Router | Track maps (MapLibre), telemetry charts (Recharts), session browser |
| **Infrastructure** | Traefik, Prometheus, Grafana | Reverse proxy, metrics, monitoring |
| **Cloud** | Cloudflare Workers + D1 | Optional cloud deployment variant |

## Performance

**Ingest** (Apple M4 Pro)
- ~31M ticks/sec parsing 10 key fields, ~2M ticks/sec for all 160+ fields
- Zero memory allocations per tick — mmap zero-copy reads
- iRacing records at 60 Hz — parser runs ~33,000x faster than real-time

**Pipeline (Old queue benchmarks)**
- 32MB / 16K record batches, worker pool defaults to CPU cores + 25%
- QuestDB writes flush at 10K rows or every second
- Memory-aware auto-pause at 5GB on Pi

Run `make bench` in `ingest/go/` for your hardware numbers.

## Getting Started

### Prerequisites

- Docker & Docker Compose (or Podman)
- Go 1.25+ (for ingest CLI)
- Node.js 18+ / pnpm (for dashboard development)

### Quick Start

```bash
cp .env.template .env
# Edit .env with your settings

# Dev mode (builds from source)
make restart-dev

# Production mode (pre-built images from GHCR)
make restart
```

### Running Services Individually

```bash
# Ingest — run on your PC where .ibt files are stored
cd ingest/go && go run ./cmd/ingest go /path/to/ibt/files

# Dashboard — local dev server
cd dashboard && pnpm install && pnpm dev

# Cloud deployment
cd cloud && npx wrangler dev
```

### Makefile Targets

| Target | Description |
|---|---|
| `make restart` | Production: pull images and start |
| `make restart-dev` | Dev: build from source and start |
| `make restart-lite` | Dev: rebuild without wiping volumes |
| `make restart-p` / `restart-dev-p` / `restart-lite-p` | Same as above with Podman |

## Project Structure

```
├── ingest/go/              # Go CLI — .ibt parsing and protobuf serialisation
├── telemetryService/
│   ├── golang/             # Go telemetry consumer and API server
│   └── telemetryService/   # C# (.NET 8) alternative consumer
├── dashboard/              # Vite + React + TanStack Router frontend
├── cloud/                  # Cloudflare Workers + D1 cloud variant
├── e2e/                    # End-to-end integration tests (Go)
├── config/                 # QuestDB, Prometheus, Grafana configs
├── traefik/                # Reverse proxy routing and TLS
├── docker-compose.yml      # Production compose
├── docker-compose.dev.yml  # Dev compose (builds from source)
└── Makefile
```

## Service Endpoints (via Traefik)

| Path | Service |
|---|---|
| `/dashboard` | Telemetry Dashboard |
| `/api` | Telemetry Service API |
| `/grafana` | Grafana |
| `/prometheus` | Prometheus |
| `/questdb` | QuestDB HTTP API |
| `/net-dash` | Traefik Dashboard |

All services accessible on both the local domain and via Tailscale (with TLS).

## Roadmap

### Now
- [ ] Remove RabbitMQ — replace with direct HTTP/gRPC ingestion
- [ ] Handle session num 0 (practice sessions)
- [ ] Track which `.ibt` files have already been ingested
- [ ] Containerise the ingest CLI for e2e testing

### Next
- [ ] ML pipeline — lap time prediction from partial lap data (XGBoost on QuestDB telemetry)
- [ ] Tire degradation curves — LSTM over stint data (lateral G, tire temps, speed, throttle)
- [ ] gRPC server streaming for live fan-out to multiple consumers
- [ ] Multicast learning implementation (PGM-style reliable UDP)

### Later
- [ ] Reinforcement learning for pit strategy optimisation
- [ ] Multi-session comparison and trend analysis
- [ ] Cloud sync between Pi and Cloudflare deployment

## License

MIT — Oliver Parkinson
