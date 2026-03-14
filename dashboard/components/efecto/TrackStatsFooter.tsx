import { useMemo } from "react";
import type { TelemetryDataPoint } from "../../lib/types";

interface TrackStatsFooterProps {
	data: TelemetryDataPoint[];
}

interface TrackStat {
	label: string;
	value: string;
	detail?: string;
}

export function TrackStatsFooter({ data }: TrackStatsFooterProps) {
	const stats = useMemo((): TrackStat[] => {
		if (data.length === 0) return [];

		const totalDistance = data.reduce(
			(sum, p) => sum + (p.distanceFromPrev || 0),
			0,
		);

		const corners = data.filter((p) => p.sectionType === "corner");
		const cornerPct = ((corners.length / data.length) * 100).toFixed(1);

		const speeds = data.map((p) => p.Speed || 0).filter((s) => s > 0);
		const minSpeed = speeds.length > 0 ? Math.min(...speeds) : 0;
		const maxSpeed = speeds.length > 0 ? Math.max(...speeds) : 0;

		return [
			{
				label: "Track Length",
				value: `${(totalDistance / 1000).toFixed(2)}`,
				detail: "km",
			},
			{
				label: "Corners",
				value: `${corners.length}`,
				detail: `pts ${cornerPct}%`,
			},
			{
				label: "Speed Range",
				value: `${minSpeed.toFixed(0)} — ${maxSpeed.toFixed(0)}`,
				detail: "km/h",
			},
			{
				label: "GPS Points",
				value: data.length.toLocaleString(),
			},
		];
	}, [data]);

	if (stats.length === 0) return null;

	return (
		<div className="flex items-stretch border-t border-border bg-background">
			{stats.map((stat) => (
				<div
					key={stat.label}
					className="flex-1 px-6 py-3 border-r border-border last:border-r-0"
				>
					<p className="text-muted-foreground text-xs mb-0.5">{stat.label}</p>
					<div className="flex items-baseline gap-1.5">
						<p className="text-foreground text-sm font-mono font-semibold">
							{stat.value}
						</p>
						{stat.detail && (
							<p className="text-muted-foreground text-xs font-mono">
								{stat.detail}
							</p>
						)}
					</div>
				</div>
			))}
		</div>
	);
}
