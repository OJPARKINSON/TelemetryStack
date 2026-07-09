import type { TelemetryDataPoint } from "./types";

interface TrackBounds {
	minLat: number;
	maxLat: number;
	minLon: number;
	maxLon: number;
}

export interface TelemetryRes {
	dataWithGPSCoordinates: null | TelemetryDataPoint[];
	trackBounds: TrackBounds | null;
	processError: string | null;
}

interface TelemetryEnvelope {
	track_name?: string;
	session_num?: string;
	points?: any[];
}

const processIRacingDataWithGPS = (
	envelope: TelemetryEnvelope | null | undefined,
): TelemetryRes => {
	const rows = envelope?.points;
	if (!rows?.length) {
		return {
			dataWithGPSCoordinates: [],
			trackBounds: null,
			processError: null,
		};
	}

	const trackName = envelope?.track_name ?? "";
	const sessionNum = envelope?.session_num ?? "";

	try {
		const points: TelemetryDataPoint[] = [];
		for (let i = 0; i < rows.length; i++) {
			const d = rows[i];
			const lat = d.lat || 0;
			const lon = d.lon || 0;
			if (!lat || !lon || Math.abs(lat) > 90 || Math.abs(lon) > 180) continue;

			points.push({
				index: points.length,
				time: d._time || points.length,
				sessionTime: d.session_time || 0,
				Speed: d.speed ? d.speed * 3.6 : 0,
				RPM: d.rpm || 0,
				Throttle: d.throttle ? d.throttle * 100 : 0,
				Brake: d.brake ? d.brake * 100 : 0,
				Gear: d.gear || 0,
				LapDistPct: d.lap_dist_pct ? d.lap_dist_pct * 100 : 0,
				SteeringWheelAngle: d.steering_wheel_angle || 0,
				Lat: lat,
				Lon: lon,
				VelocityX: d.velocity_x || 0,
				VelocityY: d.velocity_y || 0,
				VelocityZ: d.velocity_z || 0,
				FuelLevel: d.fuel_level || 0,
				LapCurrentLapTime: d.lap_current_lap_time || 0,
				PlayerCarPosition: d.player_car_position || 0,
				TrackName: trackName,
				SessionNum: sessionNum,
				LatAccel: d.lat_accel || 0,
			} as TelemetryDataPoint);
		}

		if (points.length === 0) {
			return {
				dataWithGPSCoordinates: [],
				trackBounds: null,
				processError: "No valid GPS coordinates found in telemetry data",
			};
		}

		// 5-point moving average over Lat/Lon/Speed using a running window.
		// Endpoints (first/last 2) keep their original values, matching prior behaviour.
		const n = points.length;
		const origLat = new Float64Array(n);
		const origLon = new Float64Array(n);
		const origSpeed = new Float64Array(n);
		for (let i = 0; i < n; i++) {
			origLat[i] = points[i].Lat;
			origLon[i] = points[i].Lon;
			origSpeed[i] = points[i].Speed;
		}

		if (n >= 5) {
			let sumLat = 0;
			let sumLon = 0;
			let sumSpeed = 0;
			let speedCount = 0;
			// Prime initial window [0..4]
			for (let i = 0; i < 5; i++) {
				sumLat += origLat[i];
				sumLon += origLon[i];
				if (origSpeed[i]) {
					sumSpeed += origSpeed[i];
					speedCount++;
				}
			}
			// Center index of the window starts at 2
			for (let center = 2; center <= n - 3; center++) {
				points[center].Lat = sumLat / 5;
				points[center].Lon = sumLon / 5;
				points[center].Speed =
					speedCount > 0 ? sumSpeed / speedCount : origSpeed[center];

				// Slide window: drop center-2, add center+3 (if exists)
				const out = center - 2;
				const inIdx = center + 3;
				if (inIdx < n) {
					sumLat += origLat[inIdx] - origLat[out];
					sumLon += origLon[inIdx] - origLon[out];
					if (origSpeed[out]) {
						sumSpeed -= origSpeed[out];
						speedCount--;
					}
					if (origSpeed[inIdx]) {
						sumSpeed += origSpeed[inIdx];
						speedCount++;
					}
				}
			}
		}

		return {
			dataWithGPSCoordinates: points,
			trackBounds: null,
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

export const fetcher = async (url: string): Promise<any> => {
	const controller = new AbortController();
	const timeoutId = setTimeout(() => controller.abort(), 10000);

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

export const fetcherBR = async (url: string): Promise<any> => {
	const controller = new AbortController();
	const timeoutId = setTimeout(() => controller.abort(), 10000);

	try {
		const response = await fetch(url, {
			signal: controller.signal,
			headers: {
				Accept: "application/json",
				"Accept-Encoding": "br",
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

export const telemetryFetcher = (url: string) =>
	fetch(url, {
		headers: {
			Accept: "application/json",
			"Accept-Encoding": "br",
			"Content-Type": "application/json",
		},
	}).then(async (res) => {
		return processIRacingDataWithGPS(await res.json());
	});
