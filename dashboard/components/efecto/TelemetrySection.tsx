import type { FeatureCollection } from "geojson";
import React, { useMemo, useState } from "react";

import { sectorTicks } from "../../lib/sectorTicks";
import type { TelemetryDataPoint } from "../../lib/types";
import { HoverMarker } from "../telemetry/HoverMarker";
import { MemoizedRacingLine } from "../telemetry/RacingLine";
import { SectorZoom } from "../telemetry/SectorZoom";
import type { MapStyleOption } from "../ui/map/Map";
import { NewMap as MapUI } from "../ui/map/Map";
import { MapControls } from "../ui/map/MapControls";
import { MapRoute } from "../ui/map/MapRoute";
import { SectorBar } from "./SectorBar";
import { Bar, TelemetryChartsSkeleton } from "./Skeleton";

const loadCharts = () => import("../ChartSection");
// Start the chunk download now, in parallel with the lap data request.
loadCharts();
const ChartSection = React.lazy(loadCharts);

const darkTileStyle: MapStyleOption = {
	version: 8,
	sources: {
		satellite: {
			type: "raster",
			tiles: [
				"https://cartodb-basemaps-a.global.ssl.fastly.net/dark_all/{z}/{x}/{y}.png",
			],
			tileSize: 256,
		},
	},
	layers: [
		{
			id: "satellite",
			type: "raster",
			source: "satellite",
		},
	],
};

const mapStyles = { light: darkTileStyle };

// Equal thirds for laps whose track has no sector data (ingested before the Tracks table).
const DEFAULT_SECTOR_STARTS = [0, 1 / 3, 2 / 3];

interface TelemetrySectionProps {
	dataWithGPSCoordinates: TelemetryDataPoint[];
	racingLineData: FeatureCollection | undefined;
	sectorStartPct?: number[];
	hoverCoordinates: { lon: number; lat: number } | null;
	hoveredIndex: number | null;
	onHover: (index: number | null) => void;
	onMouseLeave: () => void;
	/** First load: keep the full layout and show placeholders inside it. */
	loading?: boolean;
}

export default function TelemetrySection({
	dataWithGPSCoordinates,
	racingLineData,
	sectorStartPct: trackSectorStarts,
	hoverCoordinates,
	hoveredIndex,
	onHover,
	onMouseLeave,
	loading = false,
}: TelemetrySectionProps) {
	const sectorStartPct = trackSectorStarts ?? DEFAULT_SECTOR_STARTS;
	const [activeSector, setActiveSector] = useState<number | null>(null);

	// LapDistPct is 0-100, sector_start_pct is 0-1; the last sector ends at 100.
	const sectorRange = useMemo((): [number, number] | null => {
		const start =
			activeSector === null ? undefined : sectorStartPct[activeSector];
		if (activeSector === null || start === undefined) return null;
		return [start * 100, (sectorStartPct[activeSector + 1] ?? 1) * 100];
	}, [activeSector, sectorStartPct]);

	const ticks = useMemo(
		() => sectorTicks(dataWithGPSCoordinates, sectorStartPct),
		[dataWithGPSCoordinates, sectorStartPct],
	);

	const routeCoordinates = useMemo(
		() =>
			dataWithGPSCoordinates.map(
				(point) => [point?.Lon, point?.Lat] as [number, number],
			),
		[dataWithGPSCoordinates],
	);

	const speeds = useMemo(() => {
		const result: number[] = [];
		for (const p of dataWithGPSCoordinates) {
			const speed = p.Speed || 0;
			if (speed > 0) result.push(speed);
		}
		return result;
	}, [dataWithGPSCoordinates]);
	const minSpeed = speeds.length > 0 ? Math.min(...speeds) : 0;
	const maxSpeed = speeds.length > 0 ? Math.max(...speeds) : 0;

	const hoveredPoint =
		hoveredIndex !== null &&
		hoveredIndex >= 0 &&
		hoveredIndex < dataWithGPSCoordinates.length
			? dataWithGPSCoordinates[hoveredIndex]
			: null;

	const hasData = dataWithGPSCoordinates.length > 0;

	return (
		<div className="grid grid-cols-2 h-full w-full bg-background">
			{/* Left: Track Map */}
			<div className="flex flex-col border-r border-border flex-1 min-w-0">
				<div className="flex items-center justify-between px-6 py-3 border-b border-border">
					<div className="flex items-center gap-3">
						<p className="text-foreground text-xs font-medium">Track Map</p>
						<div className="h-3 w-px bg-border" />
						<p className="text-muted-foreground text-xs">Color by speed</p>
					</div>
					<div className="flex items-center gap-2">
						<div className="h-1.5 w-20 rounded-full bg-linear-to-r to-emerald-400 via-yellow-400 from-red-500" />
						<p className="text-muted-foreground text-xs font-mono">
							{loading ? <Bar className="w-[2ch]" /> : minSpeed.toFixed(0)}
						</p>
						<p className="text-muted-foreground text-xs">—</p>
						<p className="text-muted-foreground text-xs font-mono">
							{loading ? <Bar className="w-[3ch]" /> : maxSpeed.toFixed(0)}
						</p>
					</div>
				</div>
				<div className="flex-1 bg-card relative overflow-hidden min-h-[500px]">
					{loading ? (
						<div className="absolute inset-0 bg-secondary/40 animate-pulse" />
					) : hasData && dataWithGPSCoordinates[0]?.Lon !== undefined ? (
						<MapUI
							center={[
								dataWithGPSCoordinates[0].Lon,
								dataWithGPSCoordinates[0].Lat,
							]}
							styles={mapStyles}
							zoom={15}
						>
							<MapRoute
								coordinates={routeCoordinates}
								color="#3b82f6"
								width={0.5}
								opacity={0}
							/>
							{racingLineData && (
								<MemoizedRacingLine dataWithGPSCoordinates={racingLineData} />
							)}
							{ticks.map((coordinates, i) => (
								<MapRoute
									// biome-ignore lint/suspicious/noArrayIndexKey: ticks are positional, one per sector start
									key={i}
									id={`sector-tick-${i}`}
									coordinates={coordinates}
									color="#ffffff"
									width={2}
									opacity={0.9}
									interactive={false}
								/>
							))}
							<SectorZoom points={dataWithGPSCoordinates} range={sectorRange} />
							<MapControls showZoom showCompass />
							{hoverCoordinates && (
								<HoverMarker
									longitude={hoverCoordinates.lon}
									latitude={hoverCoordinates.lat}
								/>
							)}
						</MapUI>
					) : (
						<div className="flex items-center justify-center h-full">
							<p className="text-muted-foreground text-sm">
								No GPS data available
							</p>
						</div>
					)}
				</div>
				<SectorBar
					data={dataWithGPSCoordinates}
					sectorStartPct={sectorStartPct}
					activeSector={activeSector}
					onSectorChange={setActiveSector}
					loading={loading}
				/>
			</div>

			{/* Right: Telemetry Channels */}
			<div className="flex flex-col min-w-[400px]">
				<div className="flex items-center justify-between px-5 py-3 border-b border-border">
					<p className="text-foreground text-xs font-medium">Channels</p>
					{hoveredPoint && (
						<p className="text-muted-foreground text-xs font-mono">
							{hoveredPoint.LapDistPct?.toFixed(1)}% dist
						</p>
					)}
				</div>
				<div className="h-full overflow-y-auto px-2 py-2">
					{loading ? (
						<TelemetryChartsSkeleton />
					) : hasData ? (
						<React.Suspense fallback={<TelemetryChartsSkeleton />}>
							<ChartSection
								telemetryData={dataWithGPSCoordinates}
								onMouseLeave={onMouseLeave}
								onHover={onHover}
								xDomain={sectorRange ?? undefined}
							/>
						</React.Suspense>
					) : (
						<div className="flex items-center justify-center h-64">
							<p className="text-muted-foreground text-sm">
								No telemetry data available
							</p>
						</div>
					)}
				</div>
				<div className="flex items-center justify-between px-5 py-2 border-t border-border">
					<p className="text-muted-foreground text-xs font-mono">
						{(sectorRange?.[0] ?? 0).toFixed(0)}%
					</p>
					<p className="text-muted-foreground text-xs">Lap Distance</p>
					<p className="text-muted-foreground text-xs font-mono">
						{(sectorRange?.[1] ?? 100).toFixed(0)}%
					</p>
				</div>
			</div>
		</div>
	);
}
