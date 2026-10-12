import { LngLatBounds } from "maplibre-gl";
import { useEffect } from "react";
import type { TelemetryDataPoint } from "../../lib/types";
import { useMap } from "../ui/map/MapContext";

// Fits the map to the lap points whose LapDistPct is in range (the whole lap when null).
export function SectorZoom({
	points,
	range,
}: {
	points: TelemetryDataPoint[];
	range: [number, number] | null;
}) {
	const { map, isLoaded } = useMap();

	useEffect(() => {
		if (!isLoaded || !map) return;

		const bounds = new LngLatBounds();
		for (const p of points) {
			if (!range || (p.LapDistPct >= range[0] && p.LapDistPct <= range[1])) {
				bounds.extend([p.Lon, p.Lat]);
			}
		}
		if (bounds.isEmpty()) return;

		try {
			map.fitBounds(bounds, { padding: 40, duration: 200 });
		} catch {
			// style not ready yet
		}
	}, [isLoaded, map, points, range]);

	return null;
}
