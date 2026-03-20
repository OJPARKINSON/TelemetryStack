import type { TelemetryDataPoint } from "../../lib/types";

export function GPSAnalysisPanel({ data }: { data: TelemetryDataPoint[] }) {
	const totalDistance = data.reduce(
		(sum, point) => sum + (point.distanceFromPrev || 0),
		0,
	);
	const avgSpeed =
		data.reduce((sum, point) => sum + (point.Speed || 0), 0) / data.length;
	const maxSpeed = Math.max(...data.map((point) => point.Speed || 0));
	const minSpeed = Math.min(...data.map((point) => point.Speed || 0));

	const corners = data.filter((point) => point.sectionType === "corner");

	return (
		<div className="rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-6">
			<h2 className="mb-6 font-semibold text-lg text-white">
				GPS Track Analysis
			</h2>
			<div className="grid grid-cols-2 gap-4 md:grid-cols-4">
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/50 p-4">
					<div className="mb-2 text-sm text-zinc-400">Total Distance</div>
					<div className="font-bold text-2xl text-white">
						{(totalDistance / 1000).toFixed(2)} km
					</div>
				</div>
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/50 p-4">
					<div className="mb-2 text-sm text-zinc-400">Average Speed</div>
					<div className="font-bold text-2xl text-green-400">
						{avgSpeed.toFixed(1)} km/h
					</div>
				</div>
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/50 p-4">
					<div className="mb-2 text-sm text-zinc-400">Speed Range</div>
					<div className="font-bold text-2xl text-yellow-400">
						{minSpeed.toFixed(0)} - {maxSpeed.toFixed(0)}
					</div>
					<div className="text-xs text-zinc-500">km/h</div>
				</div>
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/50 p-4">
					<div className="mb-2 text-sm text-zinc-400">Corner Points</div>
					<div className="font-bold text-2xl text-purple-400">
						{corners.length.toLocaleString()}
					</div>
					<div className="text-xs text-zinc-500">
						{((corners.length / data.length) * 100).toFixed(1)}% of lap
					</div>
				</div>
			</div>
		</div>
	);
}
