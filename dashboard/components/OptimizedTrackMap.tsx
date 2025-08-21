"use client";

import Feature from "ol/Feature";
import { LineString, Point } from "ol/geom";
import TileLayer from "ol/layer/Tile";
import VectorLayer from "ol/layer/Vector";
import OlMap from "ol/Map";
import { fromLonLat } from "ol/proj";
import VectorSource from "ol/source/Vector";
import XYZ from "ol/source/XYZ";
import { Circle, Fill, Stroke, Style } from "ol/style";
import View from "ol/View";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { TelemetryDataPoint } from "@/lib/types";
import { usePerformanceMonitor } from "@/hooks/usePerformanceMonitor";

interface OptimizedTrackMapProps {
	dataWithCoordinates: TelemetryDataPoint[];
	selectedPointIndex: number;
	onPointClick?: (index: number) => void;
	selectedMetric?: string;
}

export default function OptimizedTrackMap({
	dataWithCoordinates,
	selectedPointIndex,
	onPointClick,
	selectedMetric = "Speed",
}: OptimizedTrackMapProps) {
	const mapRef = useRef<HTMLDivElement>(null);
	const mapInstanceRef = useRef<OlMap | null>(null);
	const racingLineSourceRef = useRef<VectorSource | null>(null);
	const markerSourceRef = useRef<VectorSource | null>(null);
	const trackRenderedRef = useRef(false);

	const [displayMetric, setDisplayMetric] = useState<string>(
		selectedMetric || "Speed",
	);

	// PERFORMANCE FIX: Color cache to avoid recalculating same colors
	const colorCacheRef = useRef<Map<string, string>>(new Map());

	// Performance monitoring (enable during development)
	const perfMonitor = usePerformanceMonitor("OptimizedTrackMap", process.env.NODE_ENV === 'development');

	// Color mapping function for different metrics with caching
	const getColorForMetric = useCallback(
		(value: number, metric: string, minVal: number, maxVal: number): string => {
			if (!value || minVal === maxVal) return "#888888";

			// PERFORMANCE FIX: Use cache to avoid recalculating same colors
			const cacheKey = `${metric}-${value}-${minVal}-${maxVal}`;
			const cached = colorCacheRef.current.get(cacheKey);
			if (cached) return cached;

			const normalized = (value - minVal) / (maxVal - minVal);
			let color: string;

			switch (metric) {
				case "Speed":
					if (normalized < 0.3) color = "#ef4444"; // Red for low speed
					else if (normalized < 0.6) color = "#f97316"; // Orange for medium
					else if (normalized < 0.8) color = "#eab308"; // Yellow for medium-high
					else color = "#22c55e"; // Green for high speed
					break;
				case "Throttle":
					color = `rgb(0, ${Math.round(150 + 105 * normalized)}, 0)`; // Green gradient
					break;
				case "Brake":
					color = `rgb(${Math.round(150 + 105 * normalized)}, 0, 0)`; // Red gradient
					break;
				case "Gear": {
					const gearColors = [
						"#ef4444",
						"#f97316",
						"#eab308",
						"#22c55e",
						"#06b6d4",
						"#8b5cf6",
						"#ec4899",
						"#f59e0b",
					];
					color = gearColors[Math.min(Math.floor(normalized * 8), 7)] || "#888888";
					break;
				}
				case "RPM":
					color = `rgb(${Math.round(255 * normalized)}, ${Math.round(100 + 155 * (1 - normalized))}, 255)`; // Purple-pink gradient
					break;
				case "SteeringWheelAngle": {
					const absNormalized = Math.abs(normalized - 0.5) * 2;
					color = `rgb(${Math.round(150 + 105 * absNormalized)}, 0, ${Math.round(150 + 105 * absNormalized)})`; // Purple for steering
					break;
				}
				default:
					color = `rgb(${Math.round(100 + 155 * normalized)}, ${Math.round(100 + 155 * (1 - normalized))}, 255)`;
			}

			// Cache the result
			colorCacheRef.current.set(cacheKey, color);
			
			// Limit cache size to prevent memory leaks
			if (colorCacheRef.current.size > 10000) {
				const firstKey = colorCacheRef.current.keys().next().value;
				colorCacheRef.current.delete(firstKey!);
			}

			return color;
		},
		[],
	);

	// Memoize the static track data - this should NEVER change once created
	const staticTrackData = useMemo(() => {
		if (!dataWithCoordinates?.length) return null;

		const validGPSPoints = dataWithCoordinates.filter(
			(point) => point.Lat && point.Lon && point.Lat !== 0 && point.Lon !== 0,
		);

		if (validGPSPoints.length === 0) return null;

		return {
			validGPSPoints,
			lineCoordinates: validGPSPoints.map((point) =>
				fromLonLat([point.Lon, point.Lat]),
			),
			bounds: {
				minLat: Math.min(...validGPSPoints.map((p) => p.Lat)),
				maxLat: Math.max(...validGPSPoints.map((p) => p.Lat)),
				minLon: Math.min(...validGPSPoints.map((p) => p.Lon)),
				maxLon: Math.max(...validGPSPoints.map((p) => p.Lon)),
			},
		};
	}, [dataWithCoordinates]); // Only recalculate if data completely changes

	// Initialize map ONCE - never again
	useEffect(() => {
		if (!mapRef.current || mapInstanceRef.current || !staticTrackData) return;

		console.log("ONE-TIME: Initializing optimized track map...");

		const racingLineSource = new VectorSource();
		const markerSource = new VectorSource();

		racingLineSourceRef.current = racingLineSource;
		markerSourceRef.current = markerSource;

		const baseLayer = new TileLayer({
			source: new XYZ({
				url: "/dashboard/api/tiles/dark/{z}/{x}/{y}",
			}),
		});

		// Create racing line layer (STATIC - track never re-renders, only colors change)
		const racingLineLayer = new VectorLayer({
			source: racingLineSource,
		});

		// Create marker layer (DYNAMIC - only this updates)
		const markerLayer = new VectorLayer({
			source: markerSource,
			style: new Style({
				image: new Circle({
					radius: 10, // 20% smaller (was 12, now 10)
					fill: new Fill({
						color: "#ffff00",
					}),
					stroke: new Stroke({
						color: "#000000",
						width: 2, // Also proportionally smaller
					}),
				}),
			}),
		});

		const map = new OlMap({
			target: mapRef.current,
			layers: [baseLayer, racingLineLayer, markerLayer],
			view: new View({
				center: fromLonLat([
					(staticTrackData.bounds.minLon + staticTrackData.bounds.maxLon) / 2,
					(staticTrackData.bounds.minLat + staticTrackData.bounds.maxLat) / 2,
				]),
				zoom: 15,
				maxZoom: 20,
				minZoom: 5,
			}),
		});

		mapInstanceRef.current = map;
		console.log("ONE-TIME: Track map initialized");

		return () => {
			if (mapInstanceRef.current) {
				mapInstanceRef.current.setTarget(undefined);
				mapInstanceRef.current = null;
			}
			racingLineSourceRef.current = null;
			markerSourceRef.current = null;
			trackRenderedRef.current = false;
		};
	}, [staticTrackData]);

	// ONE-TIME: Render the static racing line
	useEffect(() => {
		if (
			!staticTrackData ||
			!racingLineSourceRef.current ||
			trackRenderedRef.current
		)
			return;

		console.log("ONE-TIME: Rendering static racing line...");

		const source = racingLineSourceRef.current;
		source.clear(); // Clear any previous data

		// Calculate metric ranges for proper color scaling
		const metrics = [
			"Speed",
			"Throttle",
			"Brake",
			"Gear",
			"RPM",
			"SteeringWheelAngle",
		];
		const metricRanges: { [key: string]: { min: number; max: number } } = {};

		metrics.forEach((metric) => {
			const values = staticTrackData.validGPSPoints
				.map((p) => (p as any)[metric] || 0)
				.filter((v) => typeof v === "number");
			metricRanges[metric] = {
				min: Math.min(...values),
				max: Math.max(...values),
			};
		});

		// Create line segments with ALL metric data stored
		for (let i = 0; i < staticTrackData.validGPSPoints.length - 1; i++) {
			const currentPoint = staticTrackData.validGPSPoints[i];
			const nextPoint = staticTrackData.validGPSPoints[i + 1];

			const lineFeature = new Feature({
				geometry: new LineString([
					fromLonLat([currentPoint.Lon, currentPoint.Lat]),
					fromLonLat([nextPoint.Lon, nextPoint.Lat]),
				]),
				originalIndex: i,
				// Store all metric values
				Speed: currentPoint.Speed || 0,
				Throttle: currentPoint.Throttle || 0,
				Brake: currentPoint.Brake || 0,
				Gear: currentPoint.Gear || 0,
				RPM: currentPoint.RPM || 0,
				SteeringWheelAngle: currentPoint.SteeringWheelAngle || 0,
				// Store metric ranges
				Speed_min: metricRanges.Speed.min,
				Speed_max: metricRanges.Speed.max,
				Throttle_min: metricRanges.Throttle.min,
				Throttle_max: metricRanges.Throttle.max,
				Brake_min: metricRanges.Brake.min,
				Brake_max: metricRanges.Brake.max,
				Gear_min: metricRanges.Gear.min,
				Gear_max: metricRanges.Gear.max,
				RPM_min: metricRanges.RPM.min,
				RPM_max: metricRanges.RPM.max,
				SteeringWheelAngle_min: metricRanges.SteeringWheelAngle.min,
				SteeringWheelAngle_max: metricRanges.SteeringWheelAngle.max,
			});

			source.addFeature(lineFeature);
		}

		// Fit view to track bounds
		if (mapInstanceRef.current) {
			const view = mapInstanceRef.current.getView();
			view.fit(source.getExtent(), {
				padding: [50, 50, 50, 50],
				maxZoom: 18,
			});
		}

		trackRenderedRef.current = true;
		console.log(
			"ONE-TIME: Static racing line rendered with",
			staticTrackData.validGPSPoints.length,
			"points",
		);
	}, [staticTrackData]); // Only render if track data changes

	// DYNAMIC: Update ONLY the marker position with throttling
	const lastMarkerUpdateRef = useRef<number>(0);
	useEffect(() => {
		if (!markerSourceRef.current || !staticTrackData || selectedPointIndex < 0)
			return;

		const point = staticTrackData.validGPSPoints[selectedPointIndex];
		if (!point) return;

		// PERFORMANCE FIX: Throttle marker updates to 60fps max
		const now = performance.now();
		if (now - lastMarkerUpdateRef.current < 16) { // 60fps = 16.67ms
			return;
		}
		lastMarkerUpdateRef.current = now;

		// Use requestAnimationFrame for smooth marker movement
		requestAnimationFrame(() => {
			if (!markerSourceRef.current) return;

			// Clear previous marker
			markerSourceRef.current.clear();

			// Add new marker at selected position
			const markerFeature = new Feature({
				geometry: new Point(fromLonLat([point.Lon, point.Lat])),
				selectedIndex: selectedPointIndex,
			});

			markerSourceRef.current.addFeature(markerFeature);
		});

		// No console log here to avoid spam - this runs frequently
	}, [selectedPointIndex, staticTrackData]); // Updates frequently but only affects marker

	// EFFICIENT: Update only line colors when metric changes (no track re-rendering)
	useEffect(() => {
		if (!racingLineSourceRef.current || !mapInstanceRef.current) return;

		console.log("Updating racing line colors for metric:", displayMetric);

		// PERFORMANCE FIX: Clear color cache when metric changes to avoid stale colors
		colorCacheRef.current.clear();

		// Get the racing line layer
		const layers = mapInstanceRef.current.getLayers().getArray();
		const racingLineLayer = layers.find(
			(l) =>
				(l as VectorLayer<VectorSource>).getSource() ===
				racingLineSourceRef.current,
		) as VectorLayer<VectorSource>;

		if (racingLineLayer) {
			// PERFORMANCE FIX: Batch style updates using requestAnimationFrame
			const updateStyles = () => {
				const features = racingLineSourceRef.current?.getFeatures();
				if (!features) return;

				// Batch style updates to avoid layout thrashing
				features.forEach((feature) => {
					const metricValue = feature.get(displayMetric) || 0;
					const minVal = feature.get(`${displayMetric}_min`) || 0;
					const maxVal = feature.get(`${displayMetric}_max`) || 1;

					const color = getColorForMetric(
						metricValue,
						displayMetric,
						minVal,
						maxVal,
					);

					// Reuse existing style object if possible
					const existingStyle = feature.getStyle();
					if (existingStyle && typeof existingStyle !== 'function' && Array.isArray(existingStyle) === false) {
						const stroke = (existingStyle as Style).getStroke();
						if (stroke) {
							stroke.setColor(color);
							return; // Skip creating new style
						}
					}

					// Create new style only if needed
					feature.setStyle(
						new Style({
							stroke: new Stroke({
								color: color,
								width: 3,
							}),
						})
					);
				});

				// Force layer redraw
				racingLineLayer.changed();
			};

			// Use requestAnimationFrame for smooth updates
			requestAnimationFrame(updateStyles);
		}
	}, [displayMetric, getColorForMetric]);

	// Handle click events
	useEffect(() => {
		if (!mapInstanceRef.current || !onPointClick) return;

		const map = mapInstanceRef.current;

		const clickHandler = (event: any) => {
			const features = map.getFeaturesAtPixel(event.pixel);
			if (features.length > 0) {
				const feature = features[0];
				const originalIndex = feature.get("originalIndex");
				if (originalIndex !== undefined) {
					onPointClick(originalIndex);
				}
			}
		};

		map.on("click", clickHandler);
		return () => map.un("click", clickHandler);
	}, [onPointClick]);

	if (!staticTrackData) {
		return (
			<div className="h-[500px] bg-zinc-800/50 rounded-lg flex items-center justify-center">
				<div className="text-center">
					<div className="w-16 h-16 mx-auto bg-zinc-700/50 rounded-lg flex items-center justify-center mb-4">
						<div className="w-8 h-8 border-2 border-zinc-600 rounded border-dashed"></div>
					</div>
					<p className="text-zinc-400 mb-2">No GPS data available</p>
					<p className="text-zinc-500 text-sm">
						This session may not contain GPS coordinates or they may be invalid.
					</p>
				</div>
			</div>
		);
	}

	return (
		<>
			<div className=" bg-zinc-800/90 border border-zinc-600 p-3 rounded-lg text-sm z-10">
				<div className="text-white font-medium mb-2">
					{displayMetric === "Speed" && "Speed (km/h)"}
					{displayMetric === "Throttle" && "Throttle (%)"}
					{displayMetric === "Brake" && "Brake (%)"}
					{displayMetric === "Gear" && "Gear"}
					{displayMetric === "RPM" && "RPM"}
					{displayMetric === "SteeringWheelAngle" && "Steering (deg)"}
				</div>
				<div className="space-y-1">
					{displayMetric === "Speed" && (
						<>
							<div className="flex items-center space-x-2">
								<div
									className="w-3 h-3 rounded"
									style={{ backgroundColor: "#ef4444" }}
								></div>
								<span className="text-zinc-300">Low Speed</span>
							</div>
							<div className="flex items-center space-x-2">
								<div
									className="w-3 h-3 rounded"
									style={{ backgroundColor: "#eab308" }}
								></div>
								<span className="text-zinc-300">Medium Speed</span>
							</div>
							<div className="flex items-center space-x-2">
								<div
									className="w-3 h-3 rounded"
									style={{ backgroundColor: "#22c55e" }}
								></div>
								<span className="text-zinc-300">High Speed</span>
							</div>
						</>
					)}
					{displayMetric === "Throttle" && (
						<>
							<div className="flex items-center space-x-2">
								<div
									className="w-3 h-3 rounded"
									style={{ backgroundColor: "rgb(0, 150, 0)" }}
								></div>
								<span className="text-zinc-300">0% Throttle</span>
							</div>
							<div className="flex items-center space-x-2">
								<div
									className="w-3 h-3 rounded"
									style={{ backgroundColor: "rgb(0, 255, 0)" }}
								></div>
								<span className="text-zinc-300">100% Throttle</span>
							</div>
						</>
					)}
					{displayMetric === "Brake" && (
						<>
							<div className="flex items-center space-x-2">
								<div
									className="w-3 h-3 rounded"
									style={{ backgroundColor: "rgb(150, 0, 0)" }}
								></div>
								<span className="text-zinc-300">0% Brake</span>
							</div>
							<div className="flex items-center space-x-2">
								<div
									className="w-3 h-3 rounded"
									style={{ backgroundColor: "rgb(255, 0, 0)" }}
								></div>
								<span className="text-zinc-300">100% Brake</span>
							</div>
						</>
					)}
					{displayMetric === "Gear" && (
						<>
							<div className="flex items-center space-x-2">
								<div
									className="w-3 h-3 rounded"
									style={{ backgroundColor: "#ef4444" }}
								></div>
								<span className="text-zinc-300">Low Gear</span>
							</div>
							<div className="flex items-center space-x-2">
								<div
									className="w-3 h-3 rounded"
									style={{ backgroundColor: "#22c55e" }}
								></div>
								<span className="text-zinc-300">High Gear</span>
							</div>
						</>
					)}
					{displayMetric === "RPM" && (
						<>
							<div className="flex items-center space-x-2">
								<div
									className="w-3 h-3 rounded"
									style={{ backgroundColor: "#06b6d4" }}
								></div>
								<span className="text-zinc-300">Low RPM</span>
							</div>
							<div className="flex items-center space-x-2">
								<div
									className="w-3 h-3 rounded"
									style={{ backgroundColor: "#ec4899" }}
								></div>
								<span className="text-zinc-300">High RPM</span>
							</div>
						</>
					)}
					{displayMetric === "SteeringWheelAngle" && (
						<>
							<div className="flex items-center space-x-2">
								<div
									className="w-3 h-3 rounded"
									style={{ backgroundColor: "#888888" }}
								></div>
								<span className="text-zinc-300">Straight</span>
							</div>
							<div className="flex items-center space-x-2">
								<div
									className="w-3 h-3 rounded"
									style={{ backgroundColor: "#a855f7" }}
								></div>
								<span className="text-zinc-300">Full Lock</span>
							</div>
						</>
					)}
				</div>
			</div>

			{/* Controls */}
			<div className="flex flex-col space-y-2 z-10">
				{/* Zoom controls */}
				<div className="flex space-x-2">
					<button
						onClick={() => {
							if (mapInstanceRef.current) {
								const view = mapInstanceRef.current.getView();
								view.setZoom((view.getZoom() || 15) + 1);
							}
						}}
						type="button"
						className="bg-zinc-800/90 border border-zinc-600 text-white px-3 py-1 rounded text-sm font-medium hover:bg-zinc-700/90"
					>
						+
					</button>
					<button
						type="button"
						onClick={() => {
							if (mapInstanceRef.current) {
								const view = mapInstanceRef.current.getView();
								view.setZoom((view.getZoom() || 15) - 1);
							}
						}}
						className="bg-zinc-800/90 border border-zinc-600 text-white px-3 py-1 rounded text-sm font-medium hover:bg-zinc-700/90"
					>
						-
					</button>
				</div>

				{/* Metric selector */}
				<select
					value={displayMetric}
					onChange={(e) => setDisplayMetric(e.target.value)}
					className="bg-zinc-800/90 border border-zinc-600 text-white px-3 py-1 rounded text-sm font-medium hover:bg-zinc-700/90 focus:outline-none focus:ring-2 focus:ring-blue-500"
				>
					<option value="Speed">Speed</option>
					<option value="Throttle">Throttle</option>
					<option value="Brake">Brake</option>
					<option value="Gear">Gear</option>
					<option value="RPM">RPM</option>
					<option value="SteeringWheelAngle">Steering</option>
				</select>
			</div>

			<div
				ref={mapRef}
				className="w-full h-[500px] rounded-lg overflow-hidden"
			/>
		</>
	);
}
