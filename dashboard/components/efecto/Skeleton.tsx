import { CHART_CONFIGS } from "../../lib/chartConfigs";
import { cn } from "../../lib/utils";
import { CHART_GRID_CLASS, ChartCard } from "../ChartCard";

/**
 * Inline placeholder bar. Put it inside the same text element the real value
 * would use, so the line height (and therefore the layout) stays identical.
 * Size the width in `ch` inside font-mono text to match the final string.
 */
export function Bar({ className }: { className?: string }) {
	return (
		<span
			aria-hidden
			className={cn(
				"inline-block h-[0.8em] w-16 rounded-sm bg-secondary align-middle animate-pulse",
				className,
			)}
		/>
	);
}

/** The real chart cards with a pulsing box where the plot goes. */
export function TelemetryChartsSkeleton() {
	return (
		<div className={CHART_GRID_CLASS} aria-busy>
			{CHART_CONFIGS.map((config) => (
				<ChartCard key={config.dataKey} config={config}>
					<div className="h-full pb-1">
						<div className="h-full rounded bg-secondary/50 animate-pulse" />
					</div>
				</ChartCard>
			))}
		</div>
	);
}
