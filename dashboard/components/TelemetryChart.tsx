import { useCallback } from "react";
import {
	CartesianGrid,
	Line,
	LineChart,
	ReferenceLine,
	ResponsiveContainer,
	Tooltip,
	XAxis,
	YAxis,
} from "recharts";
import type { TelemetryDataPoint } from "../lib/types";
import type { chartConfig } from "./TelemetryCharts";

export const TelemetryChart = ({
	config,
	chartData,
	ReferenceLineX,
}: {
	config: chartConfig;
	chartData: TelemetryDataPoint[];
	ReferenceLineX: number;
}) => {
	const CustomTooltip = useCallback(({ active, payload }: any) => {
		if (active && payload && payload.length) {
			const dataPoint = payload[0].payload as TelemetryDataPoint & {
				lapDistance?: number;
			};
			return (
				<div className="rounded border border-zinc-600 bg-zinc-800 p-2 shadow-lg">
					<p className="text-xs text-zinc-300">
						Distance: {dataPoint.lapDistance?.toFixed(2)} km
					</p>
					<p className="text-xs text-zinc-300">
						Time: {dataPoint.sessionTime?.toFixed(2)}s
					</p>
				</div>
			);
		}
		return null;
	}, []);

	return (
		<div key={config.dataKey} className="rounded-lg bg-zinc-900/30 px-3">
			<div className="flex items-center justify-between">
				<span className="font-medium text-xs text-zinc-300">
					{config.title}
				</span>
				<span className="text-xs text-zinc-500">{config.unit}</span>
			</div>

			<div style={{ height: config.height }}>
				<ResponsiveContainer width="100%" height="100%">
					<LineChart
						data={chartData}
						onMouseMove={() => {}}
						onClick={() => {}}
						margin={{ top: 5, right: 5, left: 5, bottom: 5 }}
						syncId="telemetry-charts"
					>
						<CartesianGrid
							strokeDasharray="3 3"
							stroke="#374151"
							opacity={0.3}
						/>
						<XAxis
							dataKey="LapDistPct"
							domain={[0, 5.5]}
							type="number"
							scale="linear"
							tick={{ fill: "#9ca3af", fontSize: 10 }}
							axisLine={{ stroke: "#374151" }}
							tickLine={{ stroke: "#374151" }}
							hide
						/>
						<YAxis
							domain={config.yDomain}
							tick={{ fill: "#9ca3af", fontSize: 10 }}
							axisLine={{ stroke: "#374151" }}
							tickLine={{ stroke: "#374151" }}
							width={35}
						/>
						<Tooltip content={<CustomTooltip />} />

						<Line
							type="monotone"
							dataKey={config.dataKey}
							stroke={config.color}
							strokeWidth={1.5}
							dot={false}
							isAnimationActive={false}
							connectNulls={false}
						/>

						<ReferenceLine
							x={ReferenceLineX || 0}
							stroke="#ffffff"
							strokeWidth={1}
							strokeDasharray="2 2"
						/>
					</LineChart>
				</ResponsiveContainer>
			</div>
		</div>
	);
};
