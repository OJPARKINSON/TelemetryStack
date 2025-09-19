"use client";

import type { TelemetryDataPoint } from "../lib/types";
import OptimizedTrackMap from "./OptimizedTrackMap";

interface TrackViewProps {
	dataWithCoordinates: TelemetryDataPoint[];
	selectedPointIndex: number;
	onPointClick?: (index: number) => void;
	selectedMetric?: string;
}

export default function TrackView({
	dataWithCoordinates,
	selectedPointIndex,
	onPointClick,
	selectedMetric = "Speed",
}: TrackViewProps) {

	return (
		<OptimizedTrackMap
			dataWithCoordinates={dataWithCoordinates}
			selectedPointIndex={selectedPointIndex}
			onPointClick={onPointClick}
			selectedMetric={selectedMetric}
		/>
	);
}
