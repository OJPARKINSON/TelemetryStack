import React, { useState } from "react";
import { CHART_CONFIGS } from "../lib/chartConfigs";
import type { TelemetryDataPoint } from "../lib/types";
import { CHART_GRID_CLASS } from "./ChartCard";
import { TelemetryChart } from "./TelemetryChart";

export type { ChartConfig } from "../lib/chartConfigs";

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
	// One owner for the popover so two charts can never show it at once
	const [hoveredKey, setHoveredKey] = useState<string | null>(null);

	return (
		// biome-ignore lint/a11y/noStaticElementInteractions: na
		<div
			className={CHART_GRID_CLASS}
			onMouseLeave={() => {
				setHoveredKey(null);
				onMouseLeave?.();
			}}
		>
			{CHART_CONFIGS.map((config) => (
				<TelemetryChart
					onHover={onHover}
					showTooltip={hoveredKey === config.dataKey}
					onHoverChart={setHoveredKey}
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
