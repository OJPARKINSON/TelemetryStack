import { useCallback, useMemo, useRef, useState } from "react";
import type { TelemetryDataPoint } from "../lib/types";

/**
 * Custom hook to manage track position synchronization with chart
 */
export function useTrackPosition(telemetryData: TelemetryDataPoint[]) {
	const [selectedIndex, setSelectedIndex] = useState<number>(0);
	const [selectedLapPct, setSelectedLapPct] = useState<number>(0);

	// Cache for expensive lookups
	const lookupCacheRef = useRef<Map<number, TelemetryDataPoint>>(new Map());

	// Memoize sorted data for faster lookups
	const sortedData = useMemo(() => {
		if (!telemetryData?.length) return [];
		return [...telemetryData].sort((a, b) => a.LapDistPct - b.LapDistPct);
	}, [telemetryData]);

	/**
	 * Find the best point on the track corresponding to a specific chart index
	 * Uses a combination of methods to ensure accuracy
	 */
	const handlePointSelection = useCallback(
		(index: number) => {
			if (
				!telemetryData ||
				telemetryData.length === 0 ||
				index < 0 ||
				index >= telemetryData.length
			) {
				return;
			}

			const clickedPoint = telemetryData[index];
			setSelectedIndex(index);

			setSelectedLapPct(clickedPoint.LapDistPct);
		},
		[telemetryData],
	);

	/**
	 * Find the exact point on the track for display
	 * This ensures the marker appears at precisely the right place
	 * Optimized with caching and binary search for better performance
	 */
	const getTrackDisplayPoint = useCallback(() => {
		if (!sortedData.length) {
			return null;
		}

		// Check cache first
		const cacheKey = Math.floor(selectedLapPct * 1000); // Round to 3 decimal places for caching
		if (lookupCacheRef.current.has(cacheKey)) {
			return lookupCacheRef.current.get(cacheKey)!;
		}

		// Use binary search for faster lookup in sorted data
		let left = 0;
		let right = sortedData.length - 1;
		let bestPoint = sortedData[0];
		let minDistance = Math.abs(sortedData[0].LapDistPct - selectedLapPct);

		while (left <= right) {
			const mid = Math.floor((left + right) / 2);
			const point = sortedData[mid];
			const distance = Math.abs(point.LapDistPct - selectedLapPct);
			const wrappedDistance = Math.min(distance, 100 - distance);

			if (wrappedDistance < minDistance) {
				minDistance = wrappedDistance;
				bestPoint = point;
			}

			if (point.LapDistPct < selectedLapPct) {
				left = mid + 1;
			} else {
				right = mid - 1;
			}
		}

		// Cache the result
		lookupCacheRef.current.set(cacheKey, bestPoint);

		// Limit cache size to prevent memory leaks
		if (lookupCacheRef.current.size > 1000) {
			const firstKey = lookupCacheRef.current.keys().next().value;
			lookupCacheRef.current.delete(firstKey!);
		}

		return bestPoint;
	}, [sortedData, selectedLapPct]);

	return {
		selectedIndex,
		selectedLapPct,
		handlePointSelection,
		getTrackDisplayPoint,
	};
}
