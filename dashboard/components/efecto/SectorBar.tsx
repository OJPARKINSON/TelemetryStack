import { useMemo } from "react";
import type { TelemetryDataPoint } from "../../lib/types";

interface SectorTime {
	label: string;
	time: number;
	startPct: number;
	endPct: number;
}

interface SectorBarProps {
	data: TelemetryDataPoint[];
	sectorStartPct?: number[]; // fractions 0..1, first = 0
	activeSector: number | null; // null = full lap, 0 = S1, ...
	onSectorChange: (sector: number | null) => void;
}

const SECTOR_COLORS = ["#ef4444", "#f59e0b", "#22c55e"];

function formatSectorTime(seconds: number): string {
	if (seconds <= 0) return "--.--.--";
	return seconds.toFixed(2);
}

function useSectorTimes(
	data: TelemetryDataPoint[],
	sectorStartPct: number[] | undefined,
): SectorTime[] {
	return useMemo(() => {
		if (!sectorStartPct?.length || data.length < 10) return [];

		// Data is sorted by session_time. Find where LapCurrentLapTime
		// resets (drops significantly) — that's the true lap start.
		// After the reset, LapCurrentLapTime increases monotonically.
		let resetIndex = 0;
		for (let i = 1; i < data.length; i++) {
			if (data[i].LapCurrentLapTime < data[i - 1].LapCurrentLapTime - 10) {
				resetIndex = i;
				break;
			}
		}

		// Use only points after the reset
		const lapData = data.slice(resetIndex);
		if (lapData.length < 10) return [];

		// Sort by LapDistPct for boundary lookup
		const sorted = lapData.toSorted((a, b) => a.LapDistPct - b.LapDistPct);

		const boundaries = [...sectorStartPct.map((p) => p * 100), 100];
		const sectors: SectorTime[] = [];

		for (let i = 0; i < sectorStartPct.length; i++) {
			const startPct = boundaries[i];
			const endPct = boundaries[i + 1];

			let startPoint = sorted[0];
			let endPoint = sorted[sorted.length - 1];
			let minStartDiff = Number.POSITIVE_INFINITY;
			let minEndDiff = Number.POSITIVE_INFINITY;

			for (const p of sorted) {
				const sd = Math.abs(p.LapDistPct - startPct);
				const ed = Math.abs(p.LapDistPct - endPct);
				if (sd < minStartDiff) {
					minStartDiff = sd;
					startPoint = p;
				}
				if (ed < minEndDiff) {
					minEndDiff = ed;
					endPoint = p;
				}
			}

			const sectorTime =
				endPoint.LapCurrentLapTime - startPoint.LapCurrentLapTime;

			sectors.push({
				label: `S${i + 1}`,
				time: sectorTime > 0 ? sectorTime : 0,
				startPct,
				endPct,
			});
		}

		return sectors;
	}, [data, sectorStartPct]);
}

export function SectorBar({
	data,
	sectorStartPct,
	activeSector,
	onSectorChange,
}: SectorBarProps) {
	const sectors = useSectorTimes(data, sectorStartPct);

	if (sectors.length === 0) return null;

	return (
		<div className="flex items-center justify-between px-6 py-3 border-t border-border">
			<div className="flex items-center gap-2">
				<button
					type="button"
					onClick={() => onSectorChange(null)}
					className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
						activeSector === null
							? "bg-secondary text-foreground border border-border"
							: "text-muted-foreground hover:text-foreground"
					}`}
				>
					Full Lap
				</button>
				{sectors.map((sector, i) => (
					<button
						type="button"
						key={sector.label}
						onClick={() => onSectorChange(activeSector === i ? null : i)}
						className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
							activeSector === i
								? "bg-secondary text-foreground border border-border"
								: "text-muted-foreground hover:text-foreground"
						}`}
					>
						<span
							className="w-2 h-2 rounded-full"
							style={{
								backgroundColor: SECTOR_COLORS[i % SECTOR_COLORS.length],
							}}
						/>
						<span className="font-semibold">{sector.label}</span>
						<span className="font-mono">{formatSectorTime(sector.time)}s</span>
					</button>
				))}
			</div>
			<div className="flex items-center gap-4">
				<div className="flex items-center gap-1.5">
					<span className="w-4 h-px bg-foreground" />
					<span className="text-muted-foreground text-xs">Personal best</span>
				</div>
				<div className="flex items-center gap-1.5">
					<span className="w-4 h-px border-t border-dashed border-muted-foreground" />
					<span className="text-muted-foreground text-xs">Session best</span>
				</div>
			</div>
		</div>
	);
}
