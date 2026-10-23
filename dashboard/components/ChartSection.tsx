import React, { useState } from "react";
import { CHART_CONFIGS } from "../lib/chartConfigs";
import type { TelemetryDataPoint } from "../lib/types";
import { CHART_GRID_CLASS } from "./ChartCard";
import { TelemetryChart } from "./TelemetryChart";

interface ChartSectionProps {
	telemetryData: TelemetryDataPoint[];
	onMouseLeave?: () => void;
	onHover?: (index: number | null) => void;
}

const ChartSection = React.memo(function ChartSection({
	telemetryData,
	onMouseLeave,
	onHover,
}: ChartSectionProps) {
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

ChartSection.displayName = "ChartSection";

export default ChartSection;
