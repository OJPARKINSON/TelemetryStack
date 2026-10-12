import type { TelemetryDataPoint } from "./types";

const METRES_PER_DEG_LAT = 111_320;

// A short line across the track at each sector start, perpendicular to the direction of travel.
// sectorStartPct is 0-1; LapDistPct is 0-100.
export function sectorTicks(
	points: TelemetryDataPoint[],
	sectorStartPct: number[],
	halfLengthM = 12,
): [number, number][][] {
	if (points.length < 3) return [];

	return sectorStartPct.map((b) => {
		let i = 0;
		for (let j = 1; j < points.length; j++) {
			if (
				Math.abs(points[j].LapDistPct - b * 100) <
				Math.abs(points[i].LapDistPct - b * 100)
			)
				i = j;
		}
		const p = points[i];
		const prev = points[Math.max(i - 1, 0)];
		const next = points[Math.min(i + 1, points.length - 1)];

		// Work in metres so the tick is perpendicular on screen, not in degree space.
		const cosLat = Math.cos((p.Lat * Math.PI) / 180);
		const tx = (next.Lon - prev.Lon) * cosLat;
		const ty = next.Lat - prev.Lat;
		const len = Math.hypot(tx, ty) || 1;
		const nx = -ty / len;
		const ny = tx / len;

		const dLon = (nx * halfLengthM) / (METRES_PER_DEG_LAT * cosLat);
		const dLat = (ny * halfLengthM) / METRES_PER_DEG_LAT;
		return [
			[p.Lon - dLon, p.Lat - dLat],
			[p.Lon + dLon, p.Lat + dLat],
		];
	});
}
