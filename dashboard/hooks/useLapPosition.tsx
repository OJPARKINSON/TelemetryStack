import { useRef } from "react";
import type { TelemetryDataPoint } from "@/lib/types";

export function useLapPosition(
	telemetryData: TelemetryDataPoint[],
	_targetLapDistPct?: number,
) {
	const lastFoundIndex = useRef<number>(0);

	const findPointByLapDistPct = (lapDistPct: number): number => {
		if (!telemetryData || telemetryData.length === 0) return 0;

		const normalizedPct = lapDistPct % 100;

		let startIdx = lastFoundIndex.current;
		if (startIdx >= telemetryData.length) startIdx = 0;

		for (let i = 0; i < telemetryData.length; i++) {
			const idx = (startIdx + i) % telemetryData.length;
			if (Math.abs(telemetryData[idx].LapDistPct - normalizedPct) < 0.1) {
				lastFoundIndex.current = idx;
				return idx;
			}
		}

		let closestIdx = 0;
		let minDiff = 100;

		for (let i = 0; i < telemetryData.length; i++) {
			const diff = Math.abs(telemetryData[i].LapDistPct - normalizedPct);
			const wrapDiff = Math.min(diff, 100 - diff);

			if (wrapDiff < minDiff) {
				minDiff = wrapDiff;
				closestIdx = i;
			}
		}

		lastFoundIndex.current = closestIdx;
		return closestIdx;
	};

	const interpolatePosition = (lapDistPct: number) => {
		if (!telemetryData || telemetryData.length < 2) {
			return { index: 0, point: telemetryData?.[0] };
		}

		const normalizedPct = lapDistPct % 100;
		let beforeIdx = -1;
		let afterIdx = -1;

		const sortedPoints = [...telemetryData].sort(
			(a, b) => a.LapDistPct - b.LapDistPct,
		);

		for (let i = 0; i < sortedPoints.length; i++) {
			if (sortedPoints[i].LapDistPct <= normalizedPct) {
				beforeIdx = i;
			} else {
				afterIdx = i;
				break;
			}
		}

		if (beforeIdx === -1) {
			beforeIdx = sortedPoints.length - 1;
		}

		if (afterIdx === -1) {
			afterIdx = 0;
		}

		const beforePoint = sortedPoints[beforeIdx];
		const afterPoint = sortedPoints[afterIdx];

		let t = 0;
		if (afterPoint.LapDistPct > beforePoint.LapDistPct) {
			t =
				(normalizedPct - beforePoint.LapDistPct) /
				(afterPoint.LapDistPct - beforePoint.LapDistPct);
		} else {
			const wrappedPct =
				normalizedPct < beforePoint.LapDistPct
					? normalizedPct + 100
					: normalizedPct;
			t =
				(wrappedPct - beforePoint.LapDistPct) /
				(afterPoint.LapDistPct + 100 - beforePoint.LapDistPct);
		}

		t = Math.max(0, Math.min(1, t));

		const originalBeforeIdx = telemetryData.findIndex(
			(p) =>
				p.LapDistPct === beforePoint.LapDistPct &&
				p.sessionTime === beforePoint.sessionTime,
		);

		const originalAfterIdx = telemetryData.findIndex(
			(p) =>
				p.LapDistPct === afterPoint.LapDistPct &&
				p.sessionTime === afterPoint.sessionTime,
		);

		return {
			beforeIndex: originalBeforeIdx,
			afterIndex: originalAfterIdx,
			interpolationFactor: t,
			index: t < 0.5 ? originalBeforeIdx : originalAfterIdx,
		};
	};

	return {
		findPointByLapDistPct,
		interpolatePosition,
	};
}
