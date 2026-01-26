"use client";

import React, { useMemo } from "react";
import type { TelemetryDataPoint } from "../lib/types";
import { TelemetryChart } from "./TelemetryChart";

export type chartConfig = {
	title: string;
	dataKey: string;
	color: string;
	unit: string;
	yDomain: number[];
	height: number;
};

interface ProfessionalTelemetryChartsProps {
	telemetryData: TelemetryDataPoint[];
	selectedIndex: number;
	onHover: (index: number) => void;
	onIndexChange: (index: number) => void;
	onMouseLeave?: () => void;
}

const ProfessionalTelemetryCharts = React.memo(
	function ProfessionalTelemetryCharts({
		telemetryData,
		selectedIndex,
		onHover,
		onIndexChange,
		onMouseLeave,
	}: ProfessionalTelemetryChartsProps) {
		// const handleMouseMove = useCallback(
		// 	(data: any) => {
		// 		if (
		// 			data &&
		// 			data.activeTooltipIndex !== undefined &&
		// 			chartData.sampledData[data.activeTooltipIndex]
		// 		) {
		// 			const originalIndex =
		// 				chartData.sampledData[data.activeTooltipIndex].originalIndex;

		// 			// Skip if same index to prevent unnecessary updates
		// 			if (lastHoverIndex.current === originalIndex) return;

		// 			lastHoverIndex.current = originalIndex;
		// 			// Parent component handles throttling, so call immediately
		// 			onHover(originalIndex);
		// 		}
		// 	},
		// 	[onHover, chartData.sampledData],
		// );

		// const handleChartClick = useCallback(
		// 	(data: any) => {
		// 		if (
		// 			data &&
		// 			data.activeTooltipIndex !== undefined &&
		// 			chartData.sampledData[data.activeTooltipIndex]
		// 		) {
		// 			// Use original index from the sampled data
		// 			const originalIndex =
		// 				chartData.sampledData[data.activeTooltipIndex].originalIndex;
		// 			onIndexChange(originalIndex);
		// 		}
		// 	},
		// 	[onIndexChange, chartData.sampledData],
		// );

		// Memoize chart configurations to prevent recreations
		const chartConfigs = useMemo<chartConfig[]>(
			() => [
				{
					title: "Speed",
					dataKey: "Speed",
					color: "#ef4444",
					unit: "km/h",
					yDomain: [0, 300],
					height: 120,
				},
				{
					title: "Throttle",
					dataKey: "Throttle",
					color: "#22c55e",
					unit: "%",
					yDomain: [0, 100],
					height: 100,
				},
				{
					title: "Brake",
					dataKey: "Brake",
					color: "#f97316",
					unit: "%",
					yDomain: [0, 100],
					height: 100,
				},
				{
					title: "Gear",
					dataKey: "Gear",
					color: "#8b5cf6",
					unit: "",
					yDomain: [0, 8],
					height: 80,
				},
				{
					title: "RPM",
					dataKey: "RPM",
					color: "#06b6d4",
					unit: "",
					yDomain: [0, 8000],
					height: 100,
				},
				{
					title: "Steering",
					dataKey: "SteeringWheelAngle",
					color: "#ec4899",
					unit: "deg",
					yDomain: [-180, 180],
					height: 100,
				},
			],
			[],
		);

		return (
			// biome-ignore lint/a11y/noStaticElementInteractions: na
			<div className="flex flex-col space-y-3" onMouseLeave={onMouseLeave}>
				<div className="mb-1 font-medium text-sm text-white">
					Telemetry Data
				</div>

				{chartConfigs.map((config) => (
					<TelemetryChart
						key={config.dataKey}
						config={config}
						chartData={telemetryData}
						ReferenceLineX={
							(telemetryData?.[selectedIndex]?.LapDistPct / 100) * 5.5
						}
					/>
				))}
			</div>
		);
	},
);

ProfessionalTelemetryCharts.displayName = "ProfessionalTelemetryCharts";

export default ProfessionalTelemetryCharts;
