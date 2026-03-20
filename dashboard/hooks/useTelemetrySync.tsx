import { useEffect, useRef, useState } from "react";
import type { TelemetryDataPoint } from "../lib/types";

/**
 * Hook to synchronize chart selection with track position
 * Ensures that the selected point properly corresponds to the right position on track
 */
export function useTelemetrySync(
	telemetryData: TelemetryDataPoint[],
	selectedIndex: number,
	onIndexChange: (index: number) => void,
) {
	const [syncedIndex, setSyncedIndex] = useState<number>(selectedIndex);
	const [isProcessingSync, setIsProcessingSync] = useState<boolean>(false);
	const previousSelectionRef = useRef<number>(-1);

	/**
	 * Find the best matching point for the selected speed/position
	 * This helps ensure that when a user clicks on a low speed point in the chart,
	 * we find the corresponding position on the track that represents that point
	 */
	const findCorrespondingPoint = (index: number): number => {
		if (
			!telemetryData ||
			telemetryData.length === 0 ||
			index < 0 ||
			index >= telemetryData.length
		) {
			return index;
		}

		const selectedPoint = telemetryData[index];

		// If the selected point has a speed indicating it's a corner point,
		// make sure we're showing a position that's actually in the corner
		if (selectedPoint) {
			const maxSpeed = Math.max(...telemetryData.map((p) => p.Speed));
			const isLowSpeed = selectedPoint.Speed < maxSpeed * 0.5;

			// If it's a low-speed point (likely a corner)
			if (isLowSpeed) {
				// Look for nearby points with similar lap distance and similar speed profile
				// First find all points within 5% lap distance
				const lapDistPct = selectedPoint.LapDistPct;
				const nearbyPoints = telemetryData.filter(
					(p) =>
						Math.abs(p.LapDistPct - lapDistPct) < 5 ||
						Math.abs(p.LapDistPct - lapDistPct) > 95, // Handle wraparound near 0/100
				);

				// Find the lowest speed point in that section
				if (nearbyPoints.length > 0) {
					const lowestSpeedPoint = nearbyPoints.reduce(
						(lowest, current) =>
							current.Speed < lowest.Speed ? current : lowest,
						nearbyPoints[0],
					);

					// Return the index of this point
					const bestIndex = telemetryData.findIndex(
						(p) => p.sessionTime === lowestSpeedPoint.sessionTime,
					);

					return bestIndex >= 0 ? bestIndex : index;
				}
			}
		}

		return index;
	};

	useEffect(() => {
		// Avoid unnecessary processing
		if (previousSelectionRef.current === selectedIndex || isProcessingSync) {
			return;
		}

		setIsProcessingSync(true);

		// Find the most appropriate point to display
		const betterIndex = findCorrespondingPoint(selectedIndex);
		setSyncedIndex(betterIndex);

		// If we found a better index that's different from the selected one,
		// update the parent component
		if (betterIndex !== selectedIndex) {
			onIndexChange(betterIndex);
		}

		previousSelectionRef.current = selectedIndex;
		setIsProcessingSync(false);
		// biome-ignore lint/correctness/useExhaustiveDependencies: yeah
	}, [selectedIndex, onIndexChange, isProcessingSync, findCorrespondingPoint]);

	return {
		syncedIndex,
		findCorrespondingPoint,
	};
}
