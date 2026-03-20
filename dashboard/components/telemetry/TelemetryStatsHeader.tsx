import { formatTime } from "../../lib/formatters";

interface TelemetryStatsHeaderProps {
	trackName: string;
	gpsPointCount: number;
	maxSpeed: number;
	lapTime: number | undefined;
}

export function TelemetryStatsHeader({
	trackName,
	gpsPointCount,
	maxSpeed,
	lapTime,
}: TelemetryStatsHeaderProps) {
	return (
		<div className="rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-4">
			<div className="grid grid-cols-2 md:grid-cols-4">
				<div className="text-center">
					<div className="text-xs text-zinc-500">Track</div>
					<div className="font-semibold text-lg text-white">
						{trackName || "Unknown"}
					</div>
				</div>
				<div className="text-center">
					<div className="text-xs text-zinc-500">GPS Points</div>
					<div className="font-semibold text-green-400 text-lg">
						{gpsPointCount.toLocaleString()}
					</div>
				</div>
				<div className="text-center">
					<div className="text-xs text-zinc-500">Max Speed</div>
					<div className="font-semibold text-lg text-yellow-400">
						{maxSpeed.toFixed(0)} km/h
					</div>
				</div>
				<div className="text-center">
					<div className="text-xs text-zinc-500">Lap time</div>
					<div className="font-semibold text-blue-400 text-lg">
						{lapTime ? formatTime(lapTime) : "0.00"}
					</div>
				</div>
			</div>
		</div>
	);
}
