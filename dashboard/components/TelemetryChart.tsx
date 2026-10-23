import { memo, useCallback } from "react";
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
import { formatLapTime } from "../lib/format";
import type { TelemetryDataPoint } from "../lib/types";
import { ChartCard } from "./ChartCard";
import type { ChartConfig } from "./TelemetryCharts";

interface TelemetryChartProps {
	config: ChartConfig;
	chartData: TelemetryDataPoint[];
	ReferenceLineX: number;
	onHover?: (index: number | null) => void;
	showTooltip: boolean;
	onHoverChart: (dataKey: string) => void;
}

function formatTick(value: number): string {
	return Number.isInteger(value) ? String(value) : value.toFixed(0);
}

const CHART_MARGIN = { top: 5, right: 5, left: 5, bottom: 5 };
const X_AXIS_DOMAIN: ["dataMin", "dataMax"] = ["dataMin", "dataMax"];
const Y_AXIS_TICK = { fill: "var(--muted-foreground)", fontSize: 10 };
const Y_AXIS_LINE = { stroke: "var(--border)" };

function CustomTooltip({ active, payload, config }: any) {
	if (active && payload?.length) {
		const dataPoint = payload[0].payload as TelemetryDataPoint;
		const value = payload[0].value;

		return (
			<div className="rounded border border-border bg-card p-2 shadow-lg">
				<p className="font-medium text-foreground text-xs">
					{config.title}:{" "}
					{typeof value === "number"
						? Number.isInteger(value)
							? value
							: value.toFixed(1)
						: "—"}{" "}
					{config.unit}
				</p>
				<p className="text-xs text-muted-foreground">
					Distance: {dataPoint.LapDistPct?.toFixed(1)} %
				</p>
				<p className="text-xs text-muted-foreground">
					Lap Time: {formatLapTime(dataPoint.LapCurrentLapTime)}
				</p>
			</div>
		);
	}
	return null;
}

export const TelemetryChart = memo(function TelemetryChart({
	config,
	chartData,
	ReferenceLineX,
	onHover,
	showTooltip,
	onHoverChart,
}: TelemetryChartProps) {
	const handleMouseMove = useCallback(
		(e: any) => {
			if (onHover) {
				onHover(e.activeIndex as number);
			}
		},
		[onHover],
	);

	return (
		<ChartCard config={config}>
			<ResponsiveContainer width="100%" height="100%">
				<LineChart
					data={chartData}
					margin={CHART_MARGIN}
					syncId="telemetry-charts"
					onMouseMove={handleMouseMove}
					onMouseEnter={() => onHoverChart(config.dataKey)}
				>
					<CartesianGrid
						strokeDasharray="3 3"
						stroke="var(--border)"
						opacity={0.3}
					/>
					<XAxis
						dataKey="LapDistPct"
						domain={X_AXIS_DOMAIN}
						type="number"
						scale="linear"
						hide
					/>
					<YAxis
						domain={config.yDomain}
						tick={Y_AXIS_TICK}
						axisLine={Y_AXIS_LINE}
						tickLine={Y_AXIS_LINE}
						tickFormatter={formatTick}
						width={40}
					/>
					{/* Synced charts still draw the cursor line; only the hovered one shows the popover */}
					<Tooltip
						isAnimationActive={false}
						content={
							showTooltip ? <CustomTooltip config={config} /> : () => null
						}
					/>

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
		</ChartCard>
	);
});

TelemetryChart.displayName = "TelemetryChart";
