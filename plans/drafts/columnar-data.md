---
area: ingest
opened: 2026-10-12
---
# columnar-data

## The ask

> I want to start a new area of work scoping out moving to columular data rather than the current method. Verify this and look at the work that has gone before. Also compare against atlas data server etc

## Related

- `learnings/wire-format.md` — columnar encoding measurements (shuffle+zstd ~5.7x vs row-wise proto+zstd ~3x), 2026-09-24; test files not committed
- `learnings/ordered-ingest.md`, `learnings/ordered-ingest-plan.md` — codec choice (zstd) and ordering model that columnar interacts with
- `learnings/general.md`, `learnings/multicast.md`, `docs/architecture-reference.md` — existing ATLAS mentions
- `learnings/ibt-format-and-go-parser.md` — source .ibt layout
- `30946f8` add zstd for better transport — current wire codec
- Four `telemetry.proto` copies (ingest/go, telemetryService messaging, e2e publisher, legacy C#) — a wire change touches all live ones
- No open PRs or issues in this area (open PRs are dependabot only)

## Answered

- Which layer? > the whole pipeline including and or excluding questdb
- What to verify? > That columnar still beats what we do now
- Which systems to compare? > Any top level motorsports teams use (wec, f1)

## Open questions
