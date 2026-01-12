// TrackService.ts - Dynamic approach following track shape
import Feature from "ol/Feature";
import { LineString, Point } from "ol/geom";
import type VectorSource from "ol/source/Vector";

/**
 * Creates a racing line feature based on telemetry data
 */
export function createRacingLine(
	telemetryData: Array<{
		LapDistPct: number;
		VelocityX: number;
		VelocityY: number;
		Speed: number;
	}>,
	source: VectorSource,
): Feature | null {
	if (!telemetryData || telemetryData.length === 0) {
		return null;
	}

	try {
		// Get actual track path using car position data
		const trackWidth = 1556;
		const trackHeight = 783;

		// First, determine the actual track shape using VelocityX/Y and car positions
		// For each telemetry point, extract the relevant data about position and direction
		const trackPoints = telemetryData.map((point) => {
			const lapDistPct = point.LapDistPct;
			const velocityX = point.VelocityX;
			const velocityY = point.VelocityY;
			const speed = point.Speed;

			// Calculate normalized direction vector from velocity
			const magnitude =
				Math.sqrt(velocityX * velocityX + velocityY * velocityY) || 1;
			const dirX = velocityX / magnitude;
			const dirY = velocityY / magnitude;

			return {
				lapDistPct,
				velocityX,
				velocityY,
				speed,
				dirX,
				dirY,
			};
		});

		// Sort by lap distance
		const sortedPoints = [...trackPoints].sort(
			(a, b) => a.lapDistPct - b.lapDistPct,
		);

		// Use track SVG dimensions to map racing line
		// We'll project the telemetry data onto a 2D plane based on lap distance
		const coordinates = generateRacingLineCoordinates(
			sortedPoints,
			trackWidth,
			trackHeight,
		);

		// Create line feature
		const lineFeature = new Feature({
			geometry: new LineString(coordinates),
		});

		// Add to source
		source.addFeature(lineFeature);
		return lineFeature;
	} catch (error) {
		console.error("Error creating racing line:", error);
		return null;
	}
}

/**
 * Creates a car position marker based on telemetry data
 */
export function createCarPosition(
	telemetryPoint: any,
	source: VectorSource,
): Feature | null {
	if (!telemetryPoint) {
		return null;
	}

	try {
		// Get track dimensions
		const trackWidth = 1556;
		const trackHeight = 783;

		// Calculate car position along the track
		const lapDistPct = telemetryPoint.LapDistPct;

		// Project to canvas coordinates using lap distance
		// For consistency, use the same projection logic as the racing line
		const [x, y] = projectToTrackCoordinates(
			lapDistPct,
			trackWidth,
			trackHeight,
		);

		// Create point feature
		const pointFeature = new Feature({
			geometry: new Point([x, y]),
		});

		// Add to source
		source.addFeature(pointFeature);
		return pointFeature;
	} catch (error) {
		console.error("Error creating car position:", error);
		return null;
	}
}

/**
 * Generate coordinates for racing line based on telemetry points
 */
function generateRacingLineCoordinates(
	sortedPoints: any[],
	trackWidth: number,
	trackHeight: number,
): number[][] {
	// Create a mapping from lap distance to track position
	return sortedPoints.map((point) => {
		return projectToTrackCoordinates(point.lapDistPct, trackWidth, trackHeight);
	});
}

/**
 * Project lap distance percentage to track coordinates using track dimensions
 * This follows the actual track shape similar to what's shown in Image 2
 */
function projectToTrackCoordinates(
	lapDistPct: number,
	trackWidth: number,
	trackHeight: number,
): number[] {
	// Convert percentage to 0-1 range
	const normalizedDistance = lapDistPct / 100;

	// Track dimensions and center
	const _centerX = trackWidth / 2;
	const _centerY = trackHeight / 2;

	// Calculate angle around the track (0 to 2π)
	const _angle = normalizedDistance * 2 * Math.PI;

	// Start/finish line is at the bottom-right of the track
	const startFinishX = 1115;
	const startFinishY = 768;

	// Determine where on the track this point is
	// Using a custom shape that follows Monza's actual layout
	let x: number;
	let y: number;

	if (normalizedDistance < 0.1) {
		// Start/finish straight
		const t = normalizedDistance / 0.1;
		x = startFinishX + t * (trackWidth * 0.1);
		y = startFinishY;
	} else if (normalizedDistance < 0.2) {
		// First chicane
		const t = (normalizedDistance - 0.1) / 0.1;
		x = startFinishX + trackWidth * 0.1 - t * (trackWidth * 0.05);
		y = startFinishY - t * (trackHeight * 0.2);
	} else if (normalizedDistance < 0.3) {
		// Curve after first chicane
		const t = (normalizedDistance - 0.2) / 0.1;
		x = startFinishX + trackWidth * 0.05 - t * (trackWidth * 0.1);
		y = startFinishY - trackHeight * 0.2 - t * (trackHeight * 0.15);
	} else if (normalizedDistance < 0.4) {
		// Approaching Lesmo
		const t = (normalizedDistance - 0.3) / 0.1;
		x = startFinishX - trackWidth * 0.05 - t * (trackWidth * 0.15);
		y = startFinishY - trackHeight * 0.35 - t * (trackHeight * 0.15);
	} else if (normalizedDistance < 0.5) {
		// Lesmo corners
		const t = (normalizedDistance - 0.4) / 0.1;
		x = startFinishX - trackWidth * 0.2 - t * (trackWidth * 0.15);
		y = startFinishY - trackHeight * 0.5 + t * (trackHeight * 0.1);
	} else if (normalizedDistance < 0.6) {
		// Back straight
		const t = (normalizedDistance - 0.5) / 0.1;
		x = startFinishX - trackWidth * 0.35 - t * (trackWidth * 0.15);
		y = startFinishY - trackHeight * 0.4 + t * (trackHeight * 0.1);
	} else if (normalizedDistance < 0.7) {
		// Ascari chicane
		const t = (normalizedDistance - 0.6) / 0.1;
		x = startFinishX - trackWidth * 0.5 - t * (trackWidth * 0.15);
		y = startFinishY - trackHeight * 0.3 + t * (trackHeight * 0.2);
	} else if (normalizedDistance < 0.8) {
		// Approach to Parabolica
		const t = (normalizedDistance - 0.7) / 0.1;
		x = startFinishX - trackWidth * 0.65 + t * (trackWidth * 0.1);
		y = startFinishY - trackHeight * 0.1 + t * (trackHeight * 0.05);
	} else if (normalizedDistance < 0.9) {
		// Parabolica
		const t = (normalizedDistance - 0.8) / 0.1;
		x = startFinishX - trackWidth * 0.55 + t * (trackWidth * 0.25);
		y = startFinishY - trackHeight * 0.05 + t * (trackHeight * 0.05);
	} else {
		// Final straight back to start/finish
		const t = (normalizedDistance - 0.9) / 0.1;
		x = startFinishX - trackWidth * 0.3 + t * (trackWidth * 0.3);
		y = startFinishY;
	}

	return [x, y];
}

// Helper functions for backwards compatibility
export function getTrackPoints(numPoints = 100): number[][] {
	const trackWidth = 1556;
	const trackHeight = 783;

	const points = [];
	for (let i = 0; i < numPoints; i++) {
		const lapPct = (i / numPoints) * 100;
		points.push(projectToTrackCoordinates(lapPct, trackWidth, trackHeight));
	}

	return points;
}

export function mapLapDistanceToTrackPoint(
	lapDistPct: number,
	_numPoints = 100,
): number[] {
	const trackWidth = 1556;
	const trackHeight = 783;
	return projectToTrackCoordinates(lapDistPct, trackWidth, trackHeight);
}
