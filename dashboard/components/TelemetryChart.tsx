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
import type { ChartConfig } from "./TelemetryCharts";

interface TelemetryChartProps {
	config: ChartConfig;
	chartData: TelemetryDataPoint[];
	ReferenceLineX: number;
	onHover?: (index: number | null) => void;
}

function formatTick(value: number): string {
	return Number.isInteger(value) ? String(value) : value.toFixed(0);
}

export const TelemetryChart = ({
	config,
	chartData,
	ReferenceLineX,
	onHover,
}: TelemetryChartProps) => {
	const CustomTooltip = useCallback(({ active, payload }: any) => {
		if (active && payload && payload.length) {
			const dataPoint = payload[0].payload as TelemetryDataPoint;

			return (
				<div className="rounded border border-border bg-card p-2 shadow-lg">
					<p className="text-xs text-muted-foreground">
						Distance: {dataPoint.LapDistPct?.toFixed(1)} %
					</p>
					<p className="text-xs text-muted-foreground">
						Time: {dataPoint.sessionTime?.toFixed(2)}s
					</p>
				</div>
			);
		}
		return null;
	}, []);

	return (
		<div key={config.dataKey} className="rounded-lg bg-secondary/30 px-3">
			<div className="flex items-center justify-between py-1">
				<div className="flex items-center gap-1.5">
					<span
						className="w-1 h-3 rounded-full"
						style={{ backgroundColor: config.color }}
					/>
					<span className="font-medium text-xs text-foreground">
						{config.title}
					</span>
				</div>
				<span className="text-xs text-muted-foreground">{config.unit}</span>
			</div>

			<div style={{ height: config.height }}>
				<ResponsiveContainer width="100%" height="100%">
					<LineChart
						data={chartData}
						margin={{ top: 5, right: 5, left: 5, bottom: 5 }}
						syncId="telemetry-charts"
						onMouseMove={(e) => {
							if (onHover) {
								onHover(e.activeIndex as number);
							}
						}}
					>
						<CartesianGrid
							strokeDasharray="3 3"
							stroke="var(--border)"
							opacity={0.3}
						/>
						<XAxis
							dataKey="LapDistPct"
							domain={["dataMin", "dataMax"]}
							type="number"
							scale="linear"
							hide
						/>
						<YAxis
							domain={config.yDomain}
							tick={{ fill: "var(--muted-foreground)", fontSize: 10 }}
							axisLine={{ stroke: "var(--border)" }}
							tickLine={{ stroke: "var(--border)" }}
							tickFormatter={formatTick}
							width={40}
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
							stroke="var(--foreground)"
							strokeWidth={1}
							strokeDasharray="2 2"
						/>
					</LineChart>
				</ResponsiveContainer>
			</div>
		</div>
	);
};
