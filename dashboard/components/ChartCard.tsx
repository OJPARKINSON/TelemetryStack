import type { ReactNode } from "react";
import type { ChartConfig } from "../lib/chartConfigs";

// No recharts import here: the loading skeleton uses this frame too, and it
// must stay outside the lazily loaded chart bundle.

export const CHART_GRID_CLASS = "grid h-full space-y-1";

/** Card frame (title row + plot box) shared by the real charts and the skeleton. */
export function ChartCard({
	config,
	children,
}: {
	config: ChartConfig;
	children: ReactNode;
}) {
	return (
		<div className="rounded-lg bg-secondary/30 px-3">
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
			<div className="h-[65%]">{children}</div>
		</div>
	);
}
