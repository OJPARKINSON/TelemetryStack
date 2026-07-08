import React from "react";
import { MapMarker } from "../ui/map/MapMarker";
import { MarkerContent } from "../ui/map/MarkerContent";

export const HoverMarker = React.memo(function HoverMarker({
	longitude,
	latitude,
}: {
	longitude: number;
	latitude: number;
}) {
	return (
		<MapMarker longitude={longitude} latitude={latitude}>
			<MarkerContent>
				<div className="size-4 rounded-full border-2 border-white bg-blue-500 shadow-lg" />
			</MarkerContent>
		</MapMarker>
	);
});
