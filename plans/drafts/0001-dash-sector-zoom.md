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

## Answers (2026-10-12)
> One track would be a row in a track table
> I dont have my own outline data uness it can be infered or we get it from else where
> This isn't at the splitting stage yet

## Open questions
- Where does the track outline come from: inferred from lap telemetry (e.g. Lat/Lon), or an external source?
- What is a track row's key and columns (track id, sector boundaries, outline)?
