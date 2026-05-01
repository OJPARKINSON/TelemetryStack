import React, { useMemo } from "react";
import type { TelemetryDataPoint } from "../lib/types";
import { TelemetryChart } from "./TelemetryChart";

export type ChartConfig = {
	title: string;
	dataKey: string;
	color: string;
	unit: string;
	yDomain: number[];
};

interface TelemetryChartsProps {
	telemetryData: TelemetryDataPoint[];
	onMouseLeave?: () => void;
	onHover?: (index: number | null) => void;
}

const TelemetryCharts = React.memo(function TelemetryCharts({
	telemetryData,
	onMouseLeave,
	onHover,
}: TelemetryChartsProps) {
	const chartConfigs = useMemo<ChartConfig[]>(
		() => [
			{
				title: "Speed",
				dataKey: "Speed",
				color: "#f97316",
				unit: "km/h",
				yDomain: [0, 300],
			},
			{
				title: "Throttle",
				dataKey: "Throttle",
				color: "#22c55e",
				unit: "%",
				yDomain: [0, 100],
			},
			{
				title: "Brake",
				dataKey: "Brake",
				color: "#ef4444",
				unit: "%",
				yDomain: [0, 100],
			},
			{
				title: "Gear",
				dataKey: "Gear",
				color: "#8b5cf6",
				unit: "",
				yDomain: [0, 8],
			},
			{
				title: "RPM",
				dataKey: "RPM",
				color: "#06b6d4",
				unit: "",
				yDomain: [0, 8000],
			},
			{
				title: "Steering",
				dataKey: "SteeringWheelAngle",
				color: "#ec4899",
				unit: "deg",
				yDomain: [-3, 3],
			},
		],
		[],
	);

	return (
		// biome-ignore lint/a11y/noStaticElementInteractions: na
		<div className="grid h-full space-y-1" onMouseLeave={onMouseLeave}>
			{chartConfigs.map((config) => (
				<TelemetryChart
					onHover={onHover}
					key={config.dataKey}
					config={config}
					chartData={telemetryData}
					ReferenceLineX={telemetryData?.[0]?.LapDistPct ?? 0}
				/>
			))}
		</div>
	);
});

TelemetryCharts.displayName = "TelemetryCharts";

export default TelemetryCharts;
