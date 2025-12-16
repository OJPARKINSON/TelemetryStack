import { memo, useCallback, useMemo } from "react";
import {
	CartesianGrid,
	Line,
	LineChart,
	ReferenceDot,
	ResponsiveContainer,
	Tooltip,
	XAxis,
	YAxis,
} from "recharts";
import type { TelemetryDataPoint } from "../lib/types";

interface InfoBoxProps {
	telemetryData: any[];
	lapId: string;
	selectedMetric?: string;
	setSelectedMetric?: (metric: string) => void;
}

export const InfoBox = memo(function InfoBox({
	telemetryData,
	lapId,
}: InfoBoxProps) {
	return (
		<div className="rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-6">
			<div className="mb-6 flex items-center justify-between" />
			<div className="grid grid-cols-2 gap-4 md:grid-cols-4">
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/50 p-4">
					<div className="mb-2 text-sm text-zinc-400">Lap</div>
					<div className="font-bold text-2xl text-white">{lapId}</div>
				</div>
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/50 p-4">
					<div className="mb-2 text-sm text-zinc-400">Lap Time</div>
					<div className="font-bold text-2xl text-blue-400">
						{telemetryData[telemetryData.length - 1].LapCurrentLapTime?.toFixed(
							2,
						) || "0.00"}
					</div>
				</div>
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/50 p-4">
					<div className="mb-2 text-sm text-zinc-400">Position</div>
					<div className="font-bold text-2xl text-orange-400">
						{telemetryData[telemetryData.length - 1].PlayerCarPosition || 0}
					</div>
				</div>
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/50 p-4">
					<div className="mb-2 text-sm text-zinc-400">Fuel</div>
					<div className="font-bold text-2xl text-green-400">
						{telemetryData[telemetryData.length - 1].FuelLevel?.toFixed(1) || 0}{" "}
						L
					</div>
				</div>
			</div>
		</div>
	);
});

export const TelemetryChart = ({
	selectedMetric,
	setSelectedMetric,
	availableMetrics,
	telemetryData,
	selectedIndex,
	onIndexChange,
}: {
	selectedMetric: string;
	setSelectedMetric: (metric: string) => void;
	availableMetrics: string[];
	telemetryData: TelemetryDataPoint[];
	selectedIndex: number;
	onIndexChange: (index: number) => void;
}) => {
	const chartData = useMemo(() => {
		return telemetryData.map((point, index) => ({
			...point,
			index,
		}));
	}, [telemetryData]);

	const handleChartClick = useCallback(
		(data: any) => {
			if (data && data.activeTooltipIndex !== undefined) {
				onIndexChange(data.activeTooltipIndex);
			}
		},
		[onIndexChange],
	);

	const handleMouseMove = useCallback(() => {}, []);

	const getDataValue = (point: TelemetryDataPoint, key: string): number => {
		if (key in point) {
			return (point as any)[key] as number;
		}
		return 0;
	};

	return (
		<div>
			<div className="mb-4 flex items-center justify-between">
				<h2 className="font-semibold text-lg text-white">Telemetry Data</h2>
			</div>

			<div className="h-64">
				<ResponsiveContainer width="100%" height="100%">
					<LineChart
						data={chartData}
						onClick={handleChartClick}
						onMouseMove={handleMouseMove}
					>
						<CartesianGrid strokeDasharray="3 3" opacity={0.4} />
						<XAxis
							dataKey="SessionTime"
							label={{
								value: "Session Time (s)",
								position: "insideBottom",
								offset: 0,
								fill: "#fff",
							}}
							tick={{ fill: "#fff" }}
						/>
						<YAxis tick={{ fill: "#fff" }} />
						<Tooltip
							content={({ active, payload }) => {
								if (active && payload && payload.length) {
									const dataPoint = payload[0].payload as TelemetryDataPoint;
									return (
										<div className="rounded bg-gray-700 p-2 shadow">
											<p className="text-gray-300">
												Time: {dataPoint.sessionTime.toFixed(2)}s
											</p>
											<p className="text-red-400">
												{selectedMetric}:{" "}
												{getDataValue(dataPoint, selectedMetric).toFixed(2)}
											</p>
											<p className="text-gray-300">
												LapDistPct: {dataPoint.LapDistPct.toFixed(2)}%
											</p>
											<p className="text-gray-300">
												Speed: {dataPoint.Speed.toFixed(2)}
											</p>
										</div>
									);
								}
								return;
							}}
						/>

						{/* Low speed segments highlighted */}
						{selectedMetric === "Speed" && (
							<Line
								type="monotone"
								dataKey={selectedMetric}
								stroke="#ff6565"
								dot={false}
								isAnimationActive={false}
								activeDot={{
									r: 6,
									fill: "#ff0000",
									onClick: (dotProps: any) => {
										const index = dotProps.index;
										if (typeof index === "number") {
											onIndexChange(index);
										}
									},
								}}
							/>
						)}

						{/* Regular line for other metrics */}
						{selectedMetric !== "Speed" && (
							<Line
								type="monotone"
								dataKey={selectedMetric}
								stroke="#ff6565"
								dot={false}
								isAnimationActive={false}
								activeDot={{
									r: 6,
									fill: "#ff0000",
									onClick: (dotProps: any) => {
										const index = dotProps.index;
										if (typeof index === "number") {
											onIndexChange(index);
										}
									},
								}}
							/>
						)}

						{selectedIndex !== null &&
							selectedIndex >= 0 &&
							selectedIndex < chartData.length && (
								<ReferenceDot
									x={chartData[selectedIndex].sessionTime}
									y={getDataValue(chartData[selectedIndex], selectedMetric)}
									r={8}
									fill="#00ffff"
									stroke="#000"
								/>
							)}
					</LineChart>
				</ResponsiveContainer>
			</div>

			{/* Metric info boxes with additional information */}
			<div className="mt-4 grid grid-cols-2 gap-3">
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/30 p-3">
					<p className="mb-1 text-sm text-zinc-400">Brake</p>
					<p className="font-semibold text-lg text-white">
						{telemetryData.length > 0 &&
						selectedIndex !== null &&
						selectedIndex >= 0
							? telemetryData[selectedIndex].Brake.toFixed(1)
							: "0.0"}
					</p>
				</div>
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/30 p-3">
					<p className="mb-1 text-sm text-zinc-400">LapDistPct</p>
					<p className="font-semibold text-lg text-white">
						{telemetryData.length > 0 &&
						selectedIndex !== null &&
						selectedIndex >= 0
							? telemetryData[selectedIndex].LapDistPct.toFixed(1)
							: "0.0"}
					</p>
				</div>
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/30 p-3">
					<p className="mb-1 text-sm text-zinc-400">Speed</p>
					<p className="font-semibold text-lg text-white">
						{telemetryData.length > 0 &&
						selectedIndex !== null &&
						selectedIndex >= 0
							? telemetryData[selectedIndex].Speed.toFixed(1)
							: "277.8"}
					</p>
				</div>
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/30 p-3">
					<p className="mb-1 text-sm text-zinc-400">Throttle</p>
					<p className="font-semibold text-lg text-white">
						{telemetryData.length > 0 &&
						selectedIndex !== null &&
						selectedIndex >= 0
							? telemetryData[selectedIndex].Throttle.toFixed(1)
							: "100.0"}
					</p>
				</div>
			</div>
		</div>
	);
};
