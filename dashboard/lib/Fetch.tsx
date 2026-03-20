import type { TelemetryDataPoint } from "./types";

export const processIRacingDataWithGPS = (data: any) => {
	const sortedData = [...data].sort((a, b) => {
		const timeA = a.session_time !== undefined ? a.session_time : 0;
		const timeB = b.session_time !== undefined ? b.session_time : 0;
		return timeA - timeB;
	});

	const processedData: TelemetryDataPoint[] = sortedData.map((d, i) => ({
		index: i,
		time: d._time || i,
		sessionTime: d.session_time || 0,

		Speed: d.speed ? d.speed * 3.6 : 0,

		RPM: d.rpm || 0,

		Throttle: d.throttle ? d.throttle * 100 : 0,
		Brake: d.brake ? d.brake * 100 : 0,

		Gear: d.gear || 0,

		LapDistPct: d.lap_dist_pct ? d.lap_dist_pct * 100 : 0,

		SteeringWheelAngle: d.steering_wheel_angle || 0,

		Lat: d.lat || 0,
		Lon: d.lon || 0,

		VelocityX: d.velocity_x || 0,
		VelocityY: d.velocity_y || 0,
		VelocityZ: d.velocity_z || 0,

		FuelLevel: d.fuel_level || 0,
		LapCurrentLapTime: d.lap_current_lap_time || 0,
		PlayerCarPosition: d.player_car_position || 0,
		TrackName: d.track_name || "",
		SessionNum: d.session_num || "",

		LatAccel: d.lat_accel || 0,
		LongAccel: d.long_accel || 0,
		VertAccel: d.vert_accel || 0,

		Alt: d.alt || 0,

		Pitch: d.pitch || 0,
		Roll: d.roll || 0,
		Yaw: d.yaw || 0,
		YawNorth: d.yaw_north || 0,
	}));

	return processGPSTelemetryData(processedData);
};

interface TrackBounds {
	minLat: number;
	maxLat: number;
	minLon: number;
	maxLon: number;
}

export interface TelemetryRes {
	dataWithGPSCoordinates: null | any[];
	trackBounds: TrackBounds | null;
	processError: string | null;
}

const processGPSTelemetryData = (
	telemetry: TelemetryDataPoint[],
): TelemetryRes => {
	if (!telemetry?.length) {
		return {
			dataWithGPSCoordinates: [],
			trackBounds: null,
			processError: null,
		};
	}

	try {
		const validGPSData = telemetry.filter(
			(point) =>
				point.Lat &&
				point.Lon &&
				point.Lat !== 0 &&
				point.Lon !== 0 &&
				Math.abs(point.Lat) <= 90 &&
				Math.abs(point.Lon) <= 180,
		);

		if (validGPSData.length === 0) {
			return {
				dataWithGPSCoordinates: [],
				trackBounds: null,
				processError: "No valid GPS coordinates found in telemetry data",
			};
		}

		const lats = validGPSData.map((p) => p.Lat);
		const lons = validGPSData.map((p) => p.Lon);

		const trackBounds: TrackBounds = {
			minLat: Math.min(...lats),
			maxLat: Math.max(...lats),
			minLon: Math.min(...lons),
			maxLon: Math.max(...lons),
		};

		const speeds = validGPSData.map((p) => p.Speed).filter((s) => s > 0);
		const minSpeed = Math.min(...speeds);
		const maxSpeed = Math.max(...speeds);
		const speedRange = maxSpeed - minSpeed;

		const processedData = validGPSData.map((point, index) => {
			let distanceFromPrev = 0;
			if (index > 0) {
				const prevPoint = validGPSData[index - 1];
				distanceFromPrev = calculateGPSDistance(
					prevPoint.Lat,
					prevPoint.Lon,
					point.Lat,
					point.Lon,
				);
			}

			let calculatedSpeed = point.Speed;
			if (
				!calculatedSpeed &&
				index > 0 &&
				point.sessionTime &&
				validGPSData[index - 1].sessionTime
			) {
				const timeDiff =
					point.sessionTime - validGPSData[index - 1].sessionTime;
				if (timeDiff > 0) {
					calculatedSpeed = (distanceFromPrev / timeDiff) * 3.6; // Convert m/s to km/h
				}
			}

			let heading = 0;
			if (index > 0) {
				const prevPoint = validGPSData[index - 1];
				heading = calculateBearing(
					prevPoint.Lat,
					prevPoint.Lon,
					point.Lat,
					point.Lon,
				);
			}

			const normalizedSpeed =
				speedRange > 0 ? (point.Speed - minSpeed) / speedRange : 0;

			const lateralAccel = Math.abs(point.LatAccel || 0);

			return {
				...point,
				calculatedSpeed,
				heading,
				distanceFromPrev,
				gpsValid: true,
				originalIndex: index,
				normalizedSpeed,
				lateralAccel,
			};
		});

		const smoothedData = smoothGPSData(processedData);

		const dataWithSections = detectTrackSections(smoothedData);

		return {
			dataWithGPSCoordinates: dataWithSections,
			trackBounds,
			processError: null,
		};
	} catch (error) {
		console.error("Error processing GPS telemetry data:", error);
		return {
			dataWithGPSCoordinates: [],
			trackBounds: null,
			processError: "Error processing GPS telemetry data.",
		};
	}
};

function calculateGPSDistance(
	lat1: number,
	lon1: number,
	lat2: number,
	lon2: number,
): number {
	const R = 6371000;
	const dLat = ((lat2 - lat1) * Math.PI) / 180;
	const dLon = ((lon2 - lon1) * Math.PI) / 180;
	const a =
		Math.sin(dLat / 2) * Math.sin(dLat / 2) +
		Math.cos((lat1 * Math.PI) / 180) *
			Math.cos((lat2 * Math.PI) / 180) *
			Math.sin(dLon / 2) *
			Math.sin(dLon / 2);
	const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
	return R * c;
}

function calculateBearing(
	lat1: number,
	lon1: number,
	lat2: number,
	lon2: number,
): number {
	const dLon = ((lon2 - lon1) * Math.PI) / 180;
	const lat1Rad = (lat1 * Math.PI) / 180;
	const lat2Rad = (lat2 * Math.PI) / 180;

	const y = Math.sin(dLon) * Math.cos(lat2Rad);
	const x =
		Math.cos(lat1Rad) * Math.sin(lat2Rad) -
		Math.sin(lat1Rad) * Math.cos(lat2Rad) * Math.cos(dLon);

	const bearing = (Math.atan2(y, x) * 180) / Math.PI;
	return (bearing + 360) % 360;
}

function smoothGPSData(data: any[]): any[] {
	const windowSize = 5;

	return data.map((point, index) => {
		const start = Math.max(0, index - Math.floor(windowSize / 2));
		const end = Math.min(data.length, index + Math.floor(windowSize / 2) + 1);

		const window = data.slice(start, end);

		const avgLat = window.reduce((sum, p) => sum + p.Lat, 0) / window.length;
		const avgLon = window.reduce((sum, p) => sum + p.Lon, 0) / window.length;

		const speeds = window.filter((p) => p.Speed).map((p) => p.Speed);
		const avgSpeed =
			speeds.length > 0
				? speeds.reduce((sum, s) => sum + s, 0) / speeds.length
				: point.Speed;

		return {
			...point,
			Lat: index < 2 || index > data.length - 3 ? point.Lat : avgLat,
			Lon: index < 2 || index > data.length - 3 ? point.Lon : avgLon,
			Speed: avgSpeed,
		};
	});
}

function detectTrackSections(data: any[]): any[] {
	return data.map((point, index) => {
		let sectionType = "straight";
		let turnRadius = 0;

		if (index >= 2 && index < data.length - 2) {
			const prevHeading = data[index - 2].heading;
			const nextHeading = data[index + 2].heading;

			let headingChange = Math.abs(nextHeading - prevHeading);
			if (headingChange > 180) {
				headingChange = 360 - headingChange;
			}

			if (headingChange > 15) {
				sectionType = "corner";

				const avgSpeed = point.Speed || 0;
				if (avgSpeed > 0 && headingChange > 0) {
					const timeWindow = 4;
					turnRadius =
						(avgSpeed * timeWindow) / ((headingChange * Math.PI) / 180);
				}
			} else if (headingChange > 5) {
				sectionType = "gentle_turn";
			}
		}

		return {
			...point,
			sectionType,
			turnRadius,
		};
	});
}

export const fetcher = async (url: string): Promise<any> => {
	const controller = new AbortController();
	const timeoutId = setTimeout(() => controller.abort(), 10000); // 10 second timeout

	try {
		const response = await fetch(url, {
			signal: controller.signal,
			headers: {
				Accept: "application/json",
				"Content-Type": "application/json",
			},
		});

		clearTimeout(timeoutId);

		if (!response.ok) {
			throw new Error(
				`HTTP error! status: ${response.status} - ${response.statusText}`,
			);
		}

		return await response.json();
	} catch (error) {
		clearTimeout(timeoutId);

		if (error instanceof Error) {
			if (error.name === "AbortError") {
				throw new Error("Request timeout - please try again");
			}
			throw error;
		}

		throw new Error("An unexpected error occurred");
	}
};
