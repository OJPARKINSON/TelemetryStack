import React from "react";
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
	return (
		// biome-ignore lint/a11y/noStaticElementInteractions: na
		<div className={CHART_GRID_CLASS} onMouseLeave={onMouseLeave}>
			{CHART_CONFIGS.map((config) => (
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
