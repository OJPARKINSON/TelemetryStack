"use client";

import React, { useCallback, useEffect, useMemo, useRef } from "react";
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
		const chartData = useMemo(() => {
			// Sample data for better performance - show every nth point for charts
			// This reduces 39K points to ~4K points for rendering while maintaining shape
			const sampleRate = Math.max(1, Math.floor(telemetryData.length / 4000));

			// PERFORMANCE FIX: Build sampled data and index mapping efficiently
			const sampledData: (TelemetryDataPoint & {
				originalIndex: number;
				index: number;
				lapDistance: number;
			})[] = [];
			const originalToSampledIndex = new Map<number, number>();

			telemetryData.forEach((point, originalIndex) => {
				if (originalIndex % sampleRate === 0) {
					const sampledIndex = sampledData.length;
					sampledData.push({
						...point,
						originalIndex, // Store original index efficiently
						index: sampledIndex,
						lapDistance: (point.LapDistPct / 100) * 5.5, // Approximate track distance in km
					});
					originalToSampledIndex.set(originalIndex, sampledIndex);
				}
			});

			return { sampledData, originalToSampledIndex };
		}, [telemetryData]);

		// Throttle hover events for better performance
		const throttledHoverRef = useRef<number | null>(null);
		const lastHoverIndex = useRef<number>(-1);

		// Cleanup throttled events on unmount
		useEffect(() => {
			return () => {
				if (throttledHoverRef.current) {
					cancelAnimationFrame(throttledHoverRef.current);
				}
			};
		}, []);

		const handleMouseMove = useCallback(
			(data: any) => {
				if (
					data &&
					data.activeTooltipIndex !== undefined &&
					chartData.sampledData[data.activeTooltipIndex]
				) {
					const originalIndex =
						chartData.sampledData[data.activeTooltipIndex].originalIndex;

					// Skip if same index to prevent unnecessary updates
					if (lastHoverIndex.current === originalIndex) return;

					// Throttle hover events to 30fps (33ms) for better performance
					if (throttledHoverRef.current) {
						cancelAnimationFrame(throttledHoverRef.current);
					}
					throttledHoverRef.current = requestAnimationFrame(() => {
						lastHoverIndex.current = originalIndex;
						onHover(originalIndex);
					});
				}
			},
			[onHover, chartData.sampledData],
		);

		const handleChartClick = useCallback(
			(data: any) => {
				if (
					data &&
					data.activeTooltipIndex !== undefined &&
					chartData.sampledData[data.activeTooltipIndex]
				) {
					// Use original index from the sampled data
					const originalIndex =
						chartData.sampledData[data.activeTooltipIndex].originalIndex;
					onIndexChange(originalIndex);
				}
			},
			[onIndexChange, chartData.sampledData],
		);

		// Memoize chart configurations to prevent recreations
		const chartConfigs = useMemo(
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

		// Memoize tooltip component
		const CustomTooltip = useCallback(({ active, payload, label }: any) => {
			if (active && payload && payload.length) {
				const dataPoint = payload[0].payload as TelemetryDataPoint & {
					lapDistance?: number;
				};
				return (
					<div className="bg-zinc-800 border border-zinc-600 p-2 rounded shadow-lg">
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

		const referenceLineDistance = useMemo(() => {
			if (selectedIndex < 0 || !telemetryData[selectedIndex]) return null;

			// Try to find exact sampled point first
			const sampledIndex = chartData.originalToSampledIndex.get(selectedIndex);
			if (sampledIndex !== undefined && chartData.sampledData[sampledIndex]) {
				return chartData.sampledData[sampledIndex].lapDistance;
			}

			// Fallback: calculate from original data
			const originalPoint = telemetryData[selectedIndex];
			if (originalPoint?.LapDistPct !== undefined) {
				return (originalPoint.LapDistPct / 100) * 5.5;
			}

			return null;
		}, [
			selectedIndex,
			chartData.originalToSampledIndex,
			chartData.sampledData,
			telemetryData,
		]);

		return (
			// biome-ignore lint/a11y/noStaticElementInteractions: na
			<div className="flex flex-col space-y-3" onMouseLeave={onMouseLeave}>
				<div className="text-sm font-medium text-white mb-2">
					Telemetry Data
				</div>

				{chartConfigs.map((config) => (
					<div key={config.dataKey} className="bg-zinc-900/30 rounded-lg p-3">
						<div className="flex justify-between items-center mb-2">
							<span className="text-xs font-medium text-zinc-300">
								{config.title}
							</span>
							<span className="text-xs text-zinc-500">{config.unit}</span>
						</div>

						<div style={{ height: config.height }}>
							<ResponsiveContainer width="100%" height="100%">
								<LineChart
									data={chartData.sampledData}
									onMouseMove={handleMouseMove}
									onClick={handleChartClick}
									margin={{ top: 5, right: 5, left: 5, bottom: 5 }}
									syncId="telemetry-charts"
								>
									<CartesianGrid
										strokeDasharray="3 3"
										stroke="#374151"
										opacity={0.3}
									/>
									<XAxis
										dataKey="lapDistance"
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

									{/* FIXED: Memoized reference line to prevent jumping */}
									{referenceLineDistance !== null && (
										<ReferenceLine
											x={referenceLineDistance}
											stroke="#ffffff"
											strokeWidth={1}
											strokeDasharray="2 2"
										/>
									)}
								</LineChart>
							</ResponsiveContainer>
						</div>
					</div>
				))}

				{/* Distance markers */}
				<div className="flex justify-between text-xs text-zinc-500 px-3 mt-2">
					<span>0 km</span>
					<span>1 km</span>
					<span>2 km</span>
					<span>3 km</span>
					<span>4 km</span>
					<span>5 km</span>
				</div>
			</div>
		);
	},
);

ProfessionalTelemetryCharts.displayName = "ProfessionalTelemetryCharts";

export default ProfessionalTelemetryCharts;
