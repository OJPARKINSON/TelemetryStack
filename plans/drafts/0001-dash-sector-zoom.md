---
area: dash
ticket: 0001
opened: 2026-10-12
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

## Related
- learnings/sectors-and-zoom.md: sectors and range-zoom investigation (2026-10-03). Sectors are LapDistPct boundaries stored per track, and one Zoom object drives everything.
- dashboard/components/efecto/SectorBar.tsx, TelemetrySection.tsx: the existing sector UI.
- telemetryService/golang/internal/adapter/questdb/schema.go: the `TelemetryTicks` schema. A new table is a one-way door.
- No overlapping plans or open PRs. The only open PRs are dependabot ones.

## Open questions
- Is a separate per-track table the right store, or is there a better way, such as deriving the track shape from an existing reference lap? (The ask raises this itself.)
- What does "track data" contain: the outline geometry, the sector boundaries, or both?
- The scope crosses areas (ingest, telemetry service, dashboard). Should it be one plan, or split into a store plan and a zoom plan?
