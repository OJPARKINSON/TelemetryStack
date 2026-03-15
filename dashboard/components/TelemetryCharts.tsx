import React, { useMemo } from "react";
import type { TelemetryDataPoint } from "../lib/types";
import { TelemetryChart } from "./TelemetryChart";

export type ChartConfig = {
	title: string;
	dataKey: string;
	color: string;
	unit: string;
	yDomain: number[];
	height: number;
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
				color: "#ef4444",
				unit: "km/h",
				yDomain: [0, 300],
				height: 60,
			},
			{
				title: "Throttle",
				dataKey: "Throttle",
				color: "#22c55e",
				unit: "%",
				yDomain: [0, 100],
				height: 60,
			},
			{
				title: "Brake",
				dataKey: "Brake",
				color: "#f97316",
				unit: "%",
				yDomain: [0, 100],
				height: 60,
			},
			{
				title: "Gear",
				dataKey: "Gear",
				color: "#8b5cf6",
				unit: "",
				yDomain: [0, 8],
				height: 50,
			},
			{
				title: "RPM",
				dataKey: "RPM",
				color: "#06b6d4",
				unit: "",
				yDomain: [0, 8000],
				height: 50,
			},
			{
				title: "Steering",
				dataKey: "SteeringWheelAngle",
				color: "#ec4899",
				unit: "deg",
				yDomain: [-3, 3],
				height: 50,
			},
		],
		[],
	);

	return (
		// biome-ignore lint/a11y/noStaticElementInteractions: na
		<div className="flex flex-col space-y-1" onMouseLeave={onMouseLeave}>
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
