// @filename: lib/trackParser.ts
// ARM-compatible version without external SVG dependencies

/**
 * Simple SVG path parser that works without external dependencies
 * This is a lightweight alternative to svg-pathdata for better ARM compatibility
 */

interface PathCommand {
	type: string;
	x?: number;
	y?: number;
	x1?: number;
	y1?: number;
	x2?: number;
	y2?: number;
}

/**
 * Parses a simple SVG path and extracts points at regular intervals
 * @param svgPath The SVG path string (d attribute)
 * @param numPoints Number of points to extract (higher for more precision)
 * @returns Array of [x,y] coordinates representing the track center line
 */
export function parseSvgPath(
	svgPath: string,
	numPoints = 200,
): [number, number][] {
	try {
		// Simple path parser for basic commands (M, L, C, Z)
		const commands = parsePathCommands(svgPath);

		if (commands.length === 0) {
			console.warn("No valid path commands found");
			return [];
		}

		// Convert commands to coordinate points
		const pathPoints = commandsToPoints(commands);

		if (pathPoints.length === 0) {
			console.warn("No points generated from path commands");
			return [];
		}

		// Resample to get the requested number of points
		return resamplePoints(pathPoints, numPoints);
	} catch (error) {
		console.error("Error parsing SVG path:", error);
		// Fallback: return a simple rectangular track
		return generateFallbackTrack(numPoints);
	}
}

/**
 * Simple path command parser
 */
function parsePathCommands(pathString: string): PathCommand[] {
	const commands: PathCommand[] = [];

	// Clean up the path string
	const cleanPath = pathString
		.replace(/,/g, " ")
		.replace(/([MmLlCcSsQqTtAaHhVvZz])/g, " $1 ")
		.replace(/\s+/g, " ")
		.trim();

	const tokens = cleanPath.split(" ").filter((token) => token.length > 0);

	let currentX = 0;
	let currentY = 0;
	let i = 0;

	while (i < tokens.length) {
		const command = tokens[i];

		switch (command.toUpperCase()) {
			case "M": // Move to
				if (i + 2 < tokens.length) {
					const x = Number.parseFloat(tokens[i + 1]);
					const y = Number.parseFloat(tokens[i + 2]);
					if (!Number.isNaN(x) && !Number.isNaN(y)) {
						currentX = command === "M" ? x : currentX + x;
						currentY = command === "M" ? y : currentY + y;
						commands.push({ type: "M", x: currentX, y: currentY });
					}
					i += 3;
				} else {
					i++;
				}
				break;

			case "L": // Line to
				if (i + 2 < tokens.length) {
					const x = Number.parseFloat(tokens[i + 1]);
					const y = Number.parseFloat(tokens[i + 2]);
					if (!Number.isNaN(x) && !Number.isNaN(y)) {
						currentX = command === "L" ? x : currentX + x;
						currentY = command === "L" ? y : currentY + y;
						commands.push({ type: "L", x: currentX, y: currentY });
					}
					i += 3;
				} else {
					i++;
				}
				break;

			case "C": // Cubic Bezier curve
				if (i + 6 < tokens.length) {
					const x1 = Number.parseFloat(tokens[i + 1]);
					const y1 = Number.parseFloat(tokens[i + 2]);
					const x2 = Number.parseFloat(tokens[i + 3]);
					const y2 = Number.parseFloat(tokens[i + 4]);
					const x = Number.parseFloat(tokens[i + 5]);
					const y = Number.parseFloat(tokens[i + 6]);

					if (
						!Number.isNaN(x1) &&
						!Number.isNaN(y1) &&
						!Number.isNaN(x2) &&
						!Number.isNaN(y2) &&
						!Number.isNaN(x) &&
						!Number.isNaN(y)
					) {
						if (command === "c") {
							// Relative coordinates
							commands.push({
								type: "C",
								x1: currentX + x1,
								y1: currentY + y1,
								x2: currentX + x2,
								y2: currentY + y2,
								x: currentX + x,
								y: currentY + y,
							});
							currentX += x;
							currentY += y;
						} else {
							// Absolute coordinates
							commands.push({ type: "C", x1, y1, x2, y2, x, y });
							currentX = x;
							currentY = y;
						}
					}
					i += 7;
				} else {
					i++;
				}
				break;

			case "Z": // Close path
				commands.push({ type: "Z" });
				i++;
				break;

			default:
				// Skip unknown commands
				i++;
				break;
		}
	}

	return commands;
}

/**
 * Convert path commands to a series of coordinate points
 */
function commandsToPoints(commands: PathCommand[]): [number, number][] {
	const points: [number, number][] = [];
	let currentX = 0;
	let currentY = 0;
	let startX = 0;
	let startY = 0;

	for (const command of commands) {
		switch (command.type) {
			case "M":
				currentX = command.x!;
				currentY = command.y!;
				startX = currentX;
				startY = currentY;
				points.push([currentX, currentY]);
				break;

			case "L":
				currentX = command.x!;
				currentY = command.y!;
				points.push([currentX, currentY]);
				break;

			case "C": {
				// Approximate cubic Bezier curve with line segments
				const curvePoints = approximateCubicBezier(
					currentX,
					currentY,
					command.x1!,
					command.y1!,
					command.x2!,
					command.y2!,
					command.x!,
					command.y!,
					10, // Number of segments
				);
				points.push(...curvePoints);
				currentX = command.x!;
				currentY = command.y!;
				break;
			}

			case "Z":
				// Close path by connecting to start
				if (currentX !== startX || currentY !== startY) {
					points.push([startX, startY]);
				}
				break;
		}
	}

	return points;
}

/**
 * Approximate a cubic Bezier curve with line segments
 */
function approximateCubicBezier(
	x0: number,
	y0: number,
	x1: number,
	y1: number,
	x2: number,
	y2: number,
	x3: number,
	y3: number,
	segments: number,
): [number, number][] {
	const points: [number, number][] = [];

	for (let i = 1; i <= segments; i++) {
		const t = i / segments;
		const mt = 1 - t;

		const x =
			mt * mt * mt * x0 +
			3 * mt * mt * t * x1 +
			3 * mt * t * t * x2 +
			t * t * t * x3;

		const y =
			mt * mt * mt * y0 +
			3 * mt * mt * t * y1 +
			3 * mt * t * t * y2 +
			t * t * t * y3;

		points.push([x, y]);
	}

	return points;
}

/**
 * Resample points to get a specific number of evenly distributed points
 */
function resamplePoints(
	points: [number, number][],
	targetCount: number,
): [number, number][] {
	if (points.length === 0) return [];
	if (points.length <= targetCount) return points;

	const result: [number, number][] = [];
	const totalLength = calculateTotalLength(points);
	const segmentLength = totalLength / (targetCount - 1);

	result.push(points[0]); // Always include first point

	let currentLength = 0;
	let targetLength = segmentLength;

	for (let i = 1; i < points.length; i++) {
		const segLength = distance(points[i - 1], points[i]);
		currentLength += segLength;

		while (currentLength >= targetLength && result.length < targetCount - 1) {
			// Interpolate point at target length
			const excess = currentLength - targetLength;
			const ratio = (segLength - excess) / segLength;

			const x = points[i - 1][0] + ratio * (points[i][0] - points[i - 1][0]);
			const y = points[i - 1][1] + ratio * (points[i][1] - points[i - 1][1]);

			result.push([x, y]);
			targetLength += segmentLength;
		}
	}

	// Always include last point if we don't have enough points
	if (result.length < targetCount) {
		result.push(points[points.length - 1]);
	}

	return result;
}

/**
 * Calculate total length of a path
 */
function calculateTotalLength(points: [number, number][]): number {
	let totalLength = 0;
	for (let i = 1; i < points.length; i++) {
		totalLength += distance(points[i - 1], points[i]);
	}
	return totalLength;
}

/**
 * Calculate distance between two points
 */
function distance(p1: [number, number], p2: [number, number]): number {
	const dx = p2[0] - p1[0];
	const dy = p2[1] - p1[1];
	return Math.sqrt(dx * dx + dy * dy);
}

/**
 * Generate a fallback rectangular track if path parsing fails
 */
function generateFallbackTrack(numPoints: number): [number, number][] {
	const points: [number, number][] = [];
	const width = 1000;
	const height = 500;
	const centerX = width / 2;
	const centerY = height / 2;

	for (let i = 0; i < numPoints; i++) {
		const angle = (i / numPoints) * 2 * Math.PI;
		const x = centerX + (width / 2 - 50) * Math.cos(angle);
		const y = centerY + (height / 2 - 50) * Math.sin(angle);
		points.push([x, y]);
	}

	return points;
}

/**
 * Maps a lap distance percentage (0-1) to a point on the track SVG path
 * @param lapDistPct Lap distance percentage (0 to 1)
 * @param trackPoints Array of [x,y] coordinates representing the track
 * @returns [x,y] coordinates for the point
 */
export function mapLapDistanceToTrackPoint(
	lapDistPct: number,
	trackPoints: [number, number][],
): [number, number] {
	if (trackPoints.length === 0) return [0, 0];

	// Ensure the percentage is between 0 and 1
	lapDistPct = Math.max(0, Math.min(1, lapDistPct));

	// Calculate the index in the points array
	const pointIndex = lapDistPct * (trackPoints.length - 1);
	const lowerIndex = Math.floor(pointIndex);
	const upperIndex = Math.min(lowerIndex + 1, trackPoints.length - 1);

	// Calculate the interpolation factor
	const factor = pointIndex - lowerIndex;

	// Get the coordinates of the lower and upper points
	const [x1, y1] = trackPoints[lowerIndex];
	const [x2, y2] = trackPoints[upperIndex];

	// Interpolate between the points
	const x = x1 + factor * (x2 - x1);
	const y = y1 + factor * (y2 - y1);

	return [x, y];
}
