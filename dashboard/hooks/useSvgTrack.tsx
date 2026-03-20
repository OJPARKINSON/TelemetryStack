import { useEffect, useRef, useState } from "react";

export function useSvgTrack() {
	const [trackPath, setTrackPath] = useState<SVGPathElement | null>(null);
	const [startFinishPosition, setStartFinishPosition] = useState<number>(0);
	const [svgLoaded, setSvgLoaded] = useState<boolean>(false);
	const [svgError, setSvgError] = useState<string | null>(null);
	const svgContainerRef = useRef<HTMLDivElement | null>(null);

	useEffect(() => {
		if (svgLoaded || !svgContainerRef.current) return;

		const loadSvg = async (): Promise<void> => {
			try {
				const response = await fetch("/track-monza.svg");
				if (!response.ok) {
					throw new Error(`Failed to load SVG: ${response.statusText}`);
				}

				const svgText = await response.text();

				const parser = new DOMParser();
				const svgDoc = parser.parseFromString(svgText, "image/svg+xml");
				const pathElement = svgDoc.getElementById(
					"track-outline",
				) as SVGPathElement | null;

				if (pathElement) {
					setTrackPath(pathElement);
					// biome-ignore lint/style/noNonNullAssertion: cool
					svgContainerRef.current!.innerHTML = svgText;
					setSvgLoaded(true);

					// Find the start/finish line position
					const startFinishLine = svgDoc.getElementById(
						"start-finish-line",
					) as SVGLineElement | null;

					if (startFinishLine) {
						// Get coordinates from the start/finish line
						const startX = Number.parseFloat(
							startFinishLine.getAttribute("x1") || "0",
						);
						const startY = Number.parseFloat(
							startFinishLine.getAttribute("y1") || "0",
						);

						// Find the closest point on the track path
						const pathLength = pathElement.getTotalLength();
						let closestPoint = 0;
						let minDistance = Number.MAX_VALUE;

						// First pass with fewer samples to get approximate location
						let step = pathLength / 100;
						for (let i = 0; i < pathLength; i += step) {
							const point = pathElement.getPointAtLength(i);
							const distance = Math.sqrt(
								(point.x - startX) ** 2 + (point.y - startY) ** 2,
							);

							if (distance < minDistance) {
								minDistance = distance;
								closestPoint = i;
							}
						}

						// Second pass with higher precision around the approximate location
						const searchRange = Math.min(pathLength / 20, 100); // Limit search range
						const minSearch = Math.max(0, closestPoint - searchRange);
						const maxSearch = Math.min(pathLength, closestPoint + searchRange);

						// Much finer step for the second pass
						step = (maxSearch - minSearch) / 1000;
						for (let i = minSearch; i <= maxSearch; i += step) {
							const point = pathElement.getPointAtLength(i);
							const distance = Math.sqrt(
								(point.x - startX) ** 2 + (point.y - startY) ** 2,
							);

							if (distance < minDistance) {
								minDistance = distance;
								closestPoint = i;
							}
						}

						setStartFinishPosition(closestPoint);
						console.log(
							`Start/finish position found at ${closestPoint}/${pathLength} (${(
								(closestPoint / pathLength) * 100
							).toFixed(2)}%)`,
						);
					} else {
						console.warn(
							"Start/finish line not found in SVG, using default position",
						);
						// Use a default position if the start/finish line isn't found
						// For Monza, this is approximately at the bottom-right of the track
						setStartFinishPosition(0.75 * pathElement.getTotalLength());
					}
				} else {
					throw new Error("Track path not found in SVG");
				}
			} catch (error) {
				console.error("Error loading SVG:", error);
				setSvgError("Failed to load track. Please refresh and try again.");
			}
		};

		loadSvg();
	}, [svgLoaded]);

	return {
		trackPath,
		startFinishPosition,
		svgLoaded,
		svgContainerRef,
		svgError,
	};
}
