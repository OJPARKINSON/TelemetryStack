export interface SessionInfo {
	id: string;
	bucket: string;
}

// Type for telemetry data point
export interface TelemetryDataPoint {
	index: number;
	time: number;
	sessionTime: number;
	Speed: number; // GPS vehicle speed (converted to km/h for display)
	RPM: number;
	Throttle: number;
	Brake: number;
	Gear: number;
	LapDistPct: number;
	SteeringWheelAngle: number; // Steering wheel angle in radians
	Lat: number; // Latitude in decimal degrees
	Lon: number; // Longitude in decimal degrees
	VelocityX: number; // X velocity in m/s
	VelocityY: number; // Y velocity in m/s
	FuelLevel: number;
	LapCurrentLapTime: number;
	PlayerCarPosition: number;
	coordinates?: [number, number]; // For storing calculated coordinates
	TrackName: string;
	SessionNum: string;

	// 🆕 ADD THESE NEW FIELDS for real iRacing data:

	// Real iRacing acceleration data (all in m/s²)
	LatAccel?: number; // Lateral acceleration (including gravity)
	LongAccel?: number; // Longitudinal acceleration (including gravity)
	VertAccel?: number; // Vertical acceleration (including gravity)
	Alt?: number; // Altitude in meters

	// Processed acceleration fields
	longitudinalAccel?: number; // Processed longitudinal acceleration
	lateralAccel?: number; // Processed lateral acceleration
	verticalAccel?: number; // Processed vertical acceleration
	totalAcceleration?: number; // Combined acceleration magnitude
	horizontalAcceleration?: number; // 2D acceleration (lat + long)

	// Enhanced GPS processing fields
	elevation?: number; // Real or processed elevation
	heading?: number; // GPS-derived heading/bearing
	distanceFromPrev?: number; // Distance from previous point (meters)
	gpsValid?: boolean; // Whether GPS coordinates are valid
	originalIndex?: number; // Original index in the telemetry array
	normalizedSpeed?: number; // Speed normalized to 0-1 range for color mapping
	normalizedAcceleration?: number; // Acceleration normalized to 0-1 range
	sectionType?: "straight" | "corner" | "gentle_turn"; // Track section classification
	turnRadius?: number; // Estimated turn radius for corners

	// Additional iRacing orientation data
	VelocityZ?: number; // Z velocity in m/s
	Pitch?: number; // Pitch orientation in radians
	Roll?: number; // Roll orientation in radians
	Yaw?: number; // Yaw orientation in radians
	YawNorth?: number; // Yaw orientation relative to north in radians
}

// Type for raw telemetry data from API
interface RawTelemetryData {
	_time?: number;
	session_time?: number;
	speed?: number;
	rpm?: number;
	throttle?: number;
	brake?: number;
	gear?: number;
	lap_dist_pct?: number;
	steering_wheel_angle?: number;
	lat?: number;
	lon?: number;
	velocity_x?: number;
	velocity_y?: number;
	fuel_level?: number;
	lap_current_lap_time?: number;
	player_car_position?: number;
	track_name?: string;
	session_num?: string;
	lat_accel?: number;
	long_accel?: number;
	vert_accel?: number;
	alt?: number;
	velocity_z?: number;
	pitch?: number;
	roll?: number;
	yaw?: number;
	yaw_north?: number;
}

// Type for telemetry response from API
export interface TelemetryResponse {
	data: RawTelemetryData[];
}

export interface TelemetryResponse {
	data: RawTelemetryData[];
}

// Type for processed telemetry response with GPS data
export interface ProcessedTelemetryResponse {
	dataWithGPSCoordinates: TelemetryDataPoint[];
	trackBounds: {
		minLat: number;
		maxLat: number;
		minLon: number;
		maxLon: number;
	} | null;
	processError: string | null;
}
