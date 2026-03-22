import React, { useMemo, useState } from "react";
import type { TelemetryDataPoint } from "../../lib/types";
import { HoverMarker } from "../telemetry/HoverMarker";
import { MemoizedRacingLine } from "../telemetry/RacingLine";
import type { MapStyleOption } from "../ui/map";
import { MapControls, MapRoute, NewMap as MapUI } from "../ui/map";
import { SectorBar } from "./SectorBar";

const ProfessionalTelemetryCharts = React.lazy(
	() => import("../TelemetryCharts"),
);

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

interface TelemetrySectionProps {
	dataWithGPSCoordinates: TelemetryDataPoint[];
	racingLineData: GeoJSON.FeatureCollection | undefined;
	hoverCoordinates: { lon: number; lat: number } | null;
	hoveredIndex: number | null;
	onHover: (index: number | null) => void;
	onMouseLeave: () => void;
}

export default function TelemetrySection({
	dataWithGPSCoordinates,
	racingLineData,
	hoverCoordinates,
	hoveredIndex,
	onHover,
	onMouseLeave,
}: TelemetrySectionProps) {
	const [activeSector, setActiveSector] = useState<number | null>(null);

	const routeCoordinates = useMemo(
		() =>
			dataWithGPSCoordinates.map(
				(point) => [point?.Lon, point?.Lat] as [number, number],
			),
		[dataWithGPSCoordinates],
	);

	const speeds = useMemo(
		() => dataWithGPSCoordinates.map((p) => p.Speed || 0).filter((s) => s > 0),
		[dataWithGPSCoordinates],
	);
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
						<div className="h-1.5 w-20 rounded-full bg-gradient-to-r to-emerald-400 via-yellow-400 from-red-500" />
						<p className="text-muted-foreground text-xs font-mono">
							{minSpeed.toFixed(0)}
						</p>
						<p className="text-muted-foreground text-xs">—</p>
						<p className="text-muted-foreground text-xs font-mono">
							{maxSpeed.toFixed(0)}
						</p>
					</div>
				</div>
				<div className="flex-1 bg-card relative overflow-hidden min-h-[500px]">
					{hasData && dataWithGPSCoordinates[0]?.Lon !== undefined ? (
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
					activeSector={activeSector}
					onSectorChange={setActiveSector}
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
				<div className="flex-1 overflow-y-auto px-2 py-2">
					{hasData ? (
						<React.Suspense
							fallback={
								<div className="flex items-center justify-center h-64">
									<p className="text-muted-foreground text-sm">
										Loading charts...
									</p>
								</div>
							}
						>
							<ProfessionalTelemetryCharts
								telemetryData={dataWithGPSCoordinates}
								onMouseLeave={onMouseLeave}
								onHover={onHover}
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
					<p className="text-muted-foreground text-xs font-mono">0%</p>
					<p className="text-muted-foreground text-xs">Lap Distance</p>
					<p className="text-muted-foreground text-xs font-mono">100%</p>
				</div>
			</div>
		</div>
	);
}
