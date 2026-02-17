import { useMemo } from "react";
import type { TelemetryRes } from "../lib/Fetch";
import type { TelemetryDataPoint } from "../lib/types";

export function useTelemetryData(
	initialTelemetryData: TelemetryRes,
	sessionId: string,
	hoveredIndex: number | null,
) {
	const dataWithGPSCoordinates = useMemo(() => {
		return initialTelemetryData?.dataWithGPSCoordinates || [];
	}, [initialTelemetryData?.dataWithGPSCoordinates]);

	const trackInfo = useMemo(() => {
		if (dataWithGPSCoordinates.length === 0) return null;

		const firstPoint = dataWithGPSCoordinates[0] as TelemetryDataPoint;
		const lastPoint = dataWithGPSCoordinates[
			dataWithGPSCoordinates.length - 1
		] as TelemetryDataPoint;
		return {
			lapTime: lastPoint?.LapCurrentLapTime,
			trackName: firstPoint?.TrackName || "Unknown Track",
			sessionNum: firstPoint?.SessionNum || sessionId,
			maxSpeed: Math.max(
				...dataWithGPSCoordinates.map((p) => p.Speed || 0),
			),
		};
	}, [dataWithGPSCoordinates, sessionId]);

	const hoverCoordinates = useMemo(() => {
		if (
			hoveredIndex === null ||
			hoveredIndex < 0 ||
			hoveredIndex >= dataWithGPSCoordinates.length
		) {
			return null;
		}
		return {
			lon: dataWithGPSCoordinates[hoveredIndex].Lon,
			lat: dataWithGPSCoordinates[hoveredIndex].Lat,
		};
	}, [dataWithGPSCoordinates, hoveredIndex]);

	return { dataWithGPSCoordinates, trackInfo, hoverCoordinates };
}
