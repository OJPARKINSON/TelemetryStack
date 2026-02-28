import React, { useEffect } from "react";
import { useMap } from "../ui/map";

function RacingLine({
	dataWithGPSCoordinates,
}: {
	dataWithGPSCoordinates: GeoJSON.FeatureCollection;
}) {
	const { map, isLoaded } = useMap();

	useEffect(() => {
		if (!isLoaded || !map || !dataWithGPSCoordinates?.features) return;

		try {
			if (!map.getSource("racing-line")) {
				map.addSource("racing-line", {
					type: "geojson",
					data: dataWithGPSCoordinates,
				});
			}

			if (!map.getLayer("racing-line-layer")) {
				map.addLayer({
					id: "racing-line-layer",
					type: "line",
					source: "racing-line",
					paint: {
						"line-color": ["get", "color"],
						"line-width": 4,
						"line-opacity": 1,
					},
					layout: {
						"line-cap": "round",
						"line-join": "round",
					},
				});
			}
		} catch {
			// style not ready yet
		}

		return () => {
			try {
				if (map.getLayer("racing-line-layer"))
					map.removeLayer("racing-line-layer");
				if (map.getSource("racing-line")) map.removeSource("racing-line");
			} catch {
				// ignore — map may already be removed
			}
		};
	}, [isLoaded, map, dataWithGPSCoordinates]);

	return null;
}

export const MemoizedRacingLine = React.memo(RacingLine);
