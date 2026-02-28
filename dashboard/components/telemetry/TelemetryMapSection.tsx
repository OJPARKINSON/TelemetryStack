import { useMemo } from "react";
import { Card } from "../ui/card";
import type { MapStyleOption } from "../ui/map";
import {
	MapControls,
	MapRoute,
	NewMap as MapUI,
} from "../ui/map";
import { HoverMarker } from "./HoverMarker";
import { MemoizedRacingLine } from "./RacingLine";

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

import type { TelemetryDataPoint } from "../../lib/types";

interface TelemetryMapSectionProps {
	dataWithGPSCoordinates: TelemetryDataPoint[];
	racingLineData: GeoJSON.FeatureCollection | undefined;
	hoverCoordinates: { lon: number; lat: number } | null;
}

export function TelemetryMapSection({
	dataWithGPSCoordinates,
	racingLineData,
	hoverCoordinates,
}: TelemetryMapSectionProps) {
	const routeCoordinates = useMemo(
		() =>
			dataWithGPSCoordinates.map((data) => [data.Lon, data.Lat] as [number, number]),
		[dataWithGPSCoordinates],
	);

	return (
		<div className="col-span-1 rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-6 lg:col-span-3">
			<Card className="h-[42vw] w-full overflow-hidden p-0">
				{dataWithGPSCoordinates[0].Lon !== undefined && (
					<MapUI
						center={[
							dataWithGPSCoordinates[0]?.Lon,
							dataWithGPSCoordinates[0]?.Lat,
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
						<MapControls
							showZoom
							showCompass
							showLocate
							showFullscreen
						/>
						{hoverCoordinates && (
							<HoverMarker
								longitude={hoverCoordinates.lon}
								latitude={hoverCoordinates.lat}
							/>
						)}
					</MapUI>
				)}
			</Card>
		</div>
	);
}
