
You said: Race Team Data Platform — Reference Architecture
# Race Team Data Platform — Reference Architecture

A reference for replicating the shape of a real motorsport team's data platform
in this project. RF/over-the-air links are out of scope; this document starts
once data has reached a wired endpoint (pit wall, sim rig, timing provider's
push feed).
Confidence is called out where it matters: some of this is well-documented
industry practice, some is inferred from interviews and vendor materials.

---

## Layer 1 — Acquisition / Decoders (per source)

Each data source has its own protocol and its own decoder service. Teams do
**not** try to unify at this layer; unification happens after normalization.
Sources, with this project's analogue in parentheses:
- **Car live telemetry** — pit-wall receiver feeds ATLAS/MoTeC server.
  *(Project: live iRacing memory-mapped feed, not the .ibt file.)*
- **Car post-session** — full-fidelity logger pulled over Ethernet when the
  car comes in.
  *(Project: the .ibt file processed by ingest/.)*
- **Timing & scoring** — Al Kamel / race-control TCP or WebSocket feed.
  Sector times, GPS positions, flags, lap counts.
  *(Project: could be synthesised from iRacing session state.)*
- **Track / weather / GPS / marshalling** — additional independent feeds.
- **Strategy inputs** — tyre data, fuel-burn models, opponent data; often
  manually entered or pulled from external services.
A decoder's only job: speak the source's native protocol, produce a
normalized message, publish it onto the bus. Nothing else — no storage, no
business logic, no fan-out.
## Layer 2 — The Bus
A broker sits between decoders and everything downstream. The choice is
driven by **replay + multi-consumer**, not raw throughput.
- **Kafka** — most common in modern team stacks; long retention, partitioned
  per session/car for ordering, replayable.
- **NATS JetStream** — lighter to operate; subjects per session, durable
  consumers, per-subject ordering. Good fit for a pet project.
- **RabbitMQ** — present in older stacks. Quorum queues + consistent-hash
  exchange + publisher confirms are required to make it behave well at this
  shape; defaults are not enough.
Messages on the bus are normalized events keyed by session + car + source.
## Layer 3 — Stores (plural)
Teams do not have "a database." They have a tiered set, each fed by its own
consumer off the bus:
- **Hot time-series store** — sub-second queries for live dashboards.
  In-memory or a TSDB tier (kdb+ historically in F1; Influx, QuestDB,
  ClickHouse increasingly elsewhere).
- **Warm session store** — engineers reviewing the last few sessions.
  Relational + columnar. *This is the tier where Toyota's multi-region
  auto-backup database sits — it is a reliability story, not a throughput
  one.*
- **Cold archive** — season / multi-season analysis. Object storage with
  columnar files (Parquet on S3 or equivalent).
- **Derived / aggregate store** — laps, stints, sectors. Computed by stream
  processors reading the bus and writing back.
The same raw event lands in multiple stores via independent consumers. None
of the consumers know about each other.
## Layer 4 — Fan-out to Client Apps
The webhook / WebSocket / SSE layer. Consumers here are human-facing and
event-driven:
- Engineer dashboards
- Strategy tools
- Driver-coach apps
- Mobile apps for sporting directors / managers
These apps subscribe to **derived** streams ("lap completed", "stint ended",
"fuel below threshold"), not the raw firehose. They read history from the
warm/hot stores and receive live updates via push.
Push protocols dominate because the consumers are event-driven UIs, not
batch processors. Webhooks are common where the consumer is another service
or a third-party app.
---
## Boundary Rules
1. Client apps talk to read-side stores and the fan-out layer. **Never** to
   the bus directly, **never** to decoders.
2. Decoders publish to the bus and do nothing else.
3. Stores are populated by consumers off the bus, not by decoders writing
   directly.
4. Derived data is computed by bus consumers and republished or written to
   its own store, never computed on read.
## Build Order (pet-project pragmatic)
Do **not** build all four layers at once. The architecture only proves
itself when the second source or second consumer is added — that is the
moment the seams justify themselves.
1. One decoder → bus → one consumer → one store → one client.
2. Add a **second source** at layer 1.
3. Add a **second consumer** at layer 3 (e.g. a derived-aggregates worker).
4. Add the fan-out layer once there is more than one client app.
## Mapping to the Current Repo
As of writing, the project collapses layers 1, 2, and 3 into a single HTTP
handler (/api/ingest) plus an in-memory channel (internal/queue). The
faithful version splits these:
- ingest/ becomes one decoder among several (the .ibt post-session
  decoder).
- A new live-telemetry decoder reads the iRacing memory-mapped feed.
- A bus (Kafka or NATS JetStream) replaces the in-memory channel.
- The current DB writer becomes one consumer of many; an aggregates
  consumer and a fan-out consumer join it.
- dashboard/ consumes from the fan-out layer (WebSocket/SSE), not from
  the ingest path.
## Open Questions to Decide Later
- Bus choice: Kafka vs NATS JetStream. Driven by retention needs and
  operational appetite.
- Per-session ordering guarantees and how strictly they are enforced.
- Auth model for sim-rig producers if they ever sit outside a trusted
  network.
- Hot-store choice: stick with the current TSDB or split hot/warm tiers.