"use client";

import Feature from "ol/Feature";
import { LineString, Point } from "ol/geom";
import TileLayer from "ol/layer/Tile";
import VectorLayer from "ol/layer/Vector";
import OlMap from "ol/Map";
import { fromLonLat } from "ol/proj";
import VectorSource from "ol/source/Vector";
import XYZ from "ol/source/XYZ";
import { Circle, Fill, Stroke, Style, Text } from "ol/style";
import View from "ol/View";
import { useEffect, useRef } from "react";
import type { TelemetryDataPoint } from "@/lib/types";

interface TrackMapProps {
	dataWithCoordinates: TelemetryDataPoint[];
	selectedPointIndex: number;
	selectedLapPct: number;
	isScrubbing: boolean;
	getTrackDisplayPoint: () => TelemetryDataPoint | null;
}

export default function GPSTrackMap({
	dataWithCoordinates,
	selectedPointIndex,
	isScrubbing,
}: TrackMapProps) {
	const mapRef = useRef<HTMLDivElement>(null);
	const mapInstanceRef = useRef<OlMap | null>(null);

	const mainLineSourceRef = useRef<VectorSource | null>(null);
	const speedSegmentsSourceRef = useRef<VectorSource | null>(null);
	const carPositionSourceRef = useRef<VectorSource | null>(null);
	const selectedMarkerSourceRef = useRef<VectorSource | null>(null);

	useEffect(() => {
		if (!mapRef.current || mapInstanceRef.current) return;

		console.log("Initializing progressive track map...");

		const mainLineSource = new VectorSource();
		const speedSegmentsSource = new VectorSource();
		const carPositionSource = new VectorSource();
		const selectedMarkerSource = new VectorSource();

		mainLineSourceRef.current = mainLineSource;
		speedSegmentsSourceRef.current = speedSegmentsSource;
		carPositionSourceRef.current = carPositionSource;
		selectedMarkerSourceRef.current = selectedMarkerSource;

		const baseLayer = new TileLayer({
			source: new XYZ({
				url: "https://tile.openstreetmap.org/{z}/{x}/{y}.png",
				attributions: "© OpenStreetMap contributors",
				cacheSize: 100,
			}),
		});

		const mainLineLayer = new VectorLayer({
			source: mainLineSource,
			style: new Style({
				stroke: new Stroke({
					color: "#ed0909",
					width: 10,
				}),
			}),
		});

		const speedSegmentsLayer = new VectorLayer({
			source: speedSegmentsSource,
		});

		const carPositionLayer = new VectorLayer({
			source: carPositionSource,
			style: new Style({
				image: new Circle({
					radius: 8,
					fill: new Fill({
						color: "#00ff00",
					}),
					stroke: new Stroke({
						color: "#000000",
						width: 2,
					}),
				}),
			}),
		});

		const selectedMarkerLayer = new VectorLayer({
			source: selectedMarkerSource,
			style: new Style({
				image: new Circle({
					radius: 10,
					fill: new Fill({
						color: "#00ffff",
					}),
					stroke: new Stroke({
						color: "#000000",
						width: 2,
					}),
				}),
			}),
		});

		const map = new OlMap({
			target: mapRef.current,
			layers: [
				baseLayer,
				mainLineLayer,
				speedSegmentsLayer,
				carPositionLayer,
				selectedMarkerLayer,
			],
			view: new View({
				center: fromLonLat([9.2808, 45.6162]),
				zoom: 15,
				maxZoom: 20,
				minZoom: 5,
			}),
		});

		mapInstanceRef.current = map;

		console.log("Progressive track map initialized");

		return () => {
			if (mapInstanceRef.current) {
				mapInstanceRef.current.setTarget(undefined);
				mapInstanceRef.current = null;
			}
			mainLineSourceRef.current = null;
			speedSegmentsSourceRef.current = null;
			carPositionSourceRef.current = null;
			selectedMarkerSourceRef.current = null;
		};
	}, []);

	// biome-ignore lint/correctness/useExhaustiveDependencies: <>
	useEffect(() => {
		if (
			!mapInstanceRef.current ||
			!mainLineSourceRef.current ||
			!speedSegmentsSourceRef.current ||
			!carPositionSourceRef.current
		) {
			return;
		}

		if (dataWithCoordinates.length === 0) {
			return;
		}

		console.log(
			"Updating racing line with",
			dataWithCoordinates.length,
			"points",
		);

		mainLineSourceRef.current.clear();
		speedSegmentsSourceRef.current.clear();
		carPositionSourceRef.current.clear();

		try {
			const validGPSPoints = dataWithCoordinates.filter(
				(point) => point.Lat && point.Lon && point.Lat !== 0 && point.Lon !== 0,
			);

			if (validGPSPoints.length === 0) {
				console.warn("No valid GPS coordinates found");
				return;
			}

			console.log("Valid GPS points:", validGPSPoints.length);

			const lineCoordinates = validGPSPoints.map((point) =>
				fromLonLat([point.Lon, point.Lat]),
			);

			const mainLineFeature = new Feature({
				geometry: new LineString(lineCoordinates),
			});

			mainLineSourceRef.current.addFeature(mainLineFeature);

			let segmentsAdded = 0;
			for (let i = 0; i < validGPSPoints.length - 1; i++) {
				const point = validGPSPoints[i];
				const nextPoint = validGPSPoints[i + 1];

				if (typeof point.Speed !== "number" || isNaN(point.Speed)) {
					console.warn(`Invalid speed at point ${i}:`, point.Speed);
					continue;
				}

				const segmentCoords = [
					fromLonLat([point.Lon, point.Lat]),
					fromLonLat([nextPoint.Lon, nextPoint.Lat]),
				];

				const segmentFeature = new Feature({
					geometry: new LineString(segmentCoords),
				});

				const speedColor = getSpeedColor(point.Speed);
				segmentFeature.setStyle(
					new Style({
						stroke: new Stroke({
							color: speedColor,
							width: 3,
						}),
					}),
				);

				speedSegmentsSourceRef.current.addFeature(segmentFeature);
				segmentsAdded++;
			}

			console.log(`Added ${segmentsAdded} speed-colored segments`);

			const startPoint = validGPSPoints[0];
			if (startPoint) {
				const carFeature = new Feature({
					geometry: new Point(fromLonLat([startPoint.Lon, startPoint.Lat])),
				});
				carPositionSourceRef.current.addFeature(carFeature);
			}

			if (!isScrubbing) {
				const geometry = mainLineFeature.getGeometry();
				if (geometry) {
					mapInstanceRef.current.getView().fit(geometry.getExtent(), {
						padding: [100, 100, 100, 100],
						duration: 1000,
						maxZoom: 18,
					});
				}
			}

			console.log("Racing line updated successfully");
		} catch (error) {
			console.error("Error updating racing line:", error);
		}
	}, [dataWithCoordinates, isScrubbing]);

	useEffect(() => {
		if (!selectedMarkerSourceRef.current || dataWithCoordinates.length === 0) {
			return;
		}

		selectedMarkerSourceRef.current.clear();

		const validIndex = Math.min(
			Math.max(0, selectedPointIndex),
			dataWithCoordinates.length - 1,
		);

		const selectedPoint = dataWithCoordinates[validIndex];

		if (selectedPoint && selectedPoint.Lat && selectedPoint.Lon) {
			const markerCoords = fromLonLat([selectedPoint.Lon, selectedPoint.Lat]);

			const markerFeature = new Feature({
				geometry: new Point(markerCoords),
			});

			const displayText = `${selectedPoint.LapDistPct.toFixed(1)}% - ${selectedPoint.Speed.toFixed(1)}kph`;

			markerFeature.setStyle(
				new Style({
					image: new Circle({
						radius: 10,
						fill: new Fill({
							color: "#00ffff",
						}),
						stroke: new Stroke({
							color: "#000000",
							width: 2,
						}),
					}),
					text: new Text({
						text: displayText,
						offsetY: -20,
						font: "14px sans-serif",
						fill: new Fill({
							color: "#ffffff",
						}),
						stroke: new Stroke({
							color: "#000000",
							width: 3,
						}),
					}),
				}),
			);

			selectedMarkerSourceRef.current.addFeature(markerFeature);

			if (isScrubbing && mapInstanceRef.current) {
				mapInstanceRef.current.getView().animate({
					center: markerCoords,
					duration: 300,
				});
			}
		}
	}, [selectedPointIndex, dataWithCoordinates, isScrubbing]);

	const getSpeedColor = (speed: number): string => {
		const normalizedSpeed = Math.min(speed / 300, 1);

		if (normalizedSpeed < 0.3) {
			return "#ff0000";
		} else if (normalizedSpeed < 0.6) {
			return "#ffff00";
		} else {
			return "#00ff00";
		}
	};

	const handleZoomIn = (): void => {
		if (mapInstanceRef.current) {
			const view = mapInstanceRef.current.getView();
			const currentZoom = view.getZoom();
			if (currentZoom && currentZoom < 20) {
				view.animate({
					zoom: currentZoom + 0.5,
					duration: 250,
				});
			}
		}
	};

	const handleZoomOut = (): void => {
		if (mapInstanceRef.current) {
			const view = mapInstanceRef.current.getView();
			const currentZoom = view.getZoom();
			if (currentZoom && currentZoom > 5) {
				view.animate({
					zoom: currentZoom - 0.5,
					duration: 250,
				});
			}
		}
	};

	return (
		<div className="h-[500px] bg-gray-800 rounded-lg relative">
			<div className="absolute top-2 left-2 z-10 flex flex-col gap-1">
				<button
					type="button"
					onClick={handleZoomIn}
					className="bg-gray-700 hover:bg-gray-600 text-white w-8 h-8 flex items-center justify-center rounded shadow"
					aria-label="Zoom in"
				>
					+
				</button>
				<button
					type="button"
					onClick={handleZoomOut}
					className="bg-gray-700 hover:bg-gray-600 text-white w-8 h-8 flex items-center justify-center rounded shadow"
					aria-label="Zoom out"
				>
					-
				</button>
			</div>

			<div className="absolute top-2 right-2 z-10 bg-gray-700 bg-opacity-90 p-2 rounded text-white text-xs">
				<div className="flex items-center gap-2 mb-1">
					<div className="w-4 h-1 bg-red-500"></div>
					<span>Low Speed</span>
				</div>
				<div className="flex items-center gap-2 mb-1">
					<div className="w-4 h-1 bg-yellow-500"></div>
					<span>Medium Speed</span>
				</div>
				<div className="flex items-center gap-2">
					<div className="w-4 h-1 bg-green-500"></div>
					<span>High Speed</span>
				</div>
			</div>

			<div className="absolute bottom-2 left-2 z-10 bg-gray-700 bg-opacity-90 p-2 rounded text-white text-xs">
				<div>GPS Points: {dataWithCoordinates.length}</div>
				<div>Map: {mapInstanceRef.current ? "Initialized" : "Not ready"}</div>
				<div>
					Main Line: {mainLineSourceRef.current?.getFeatures().length || 0}
				</div>
				<div>
					Speed Segments:{" "}
					{speedSegmentsSourceRef.current?.getFeatures().length || 0}
				</div>
			</div>

			<div
				ref={mapRef}
				className="w-full h-full"
				style={{
					width: "100%",
					height: "100%",
				}}
			/>
		</div>
	);
}
