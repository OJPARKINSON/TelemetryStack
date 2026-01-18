import { Link, useNavigate } from "@tanstack/react-router";
import React, {
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import useSWR from "swr";
import { InfoBox } from "../components/InfoBox";
import { Card } from "../components/ui/card";
// biome-ignore lint/suspicious/noShadowRestrictedNames: <explanation>
import { Map, MapControls, MapRoute, useMap } from "../components/ui/map";
import { useTrackPosition } from "../hooks/useTrackPosition";
import { fetcher, type TelemetryRes } from "../lib/Fetch";
import type { TelemetryDataPoint } from "../lib/types";

const ProfessionalTelemetryCharts = React.lazy(
	() => import("./ProfessionalTelemetryCharts"),
);

interface TelemetryPageProps {
	initialTelemetryData: TelemetryRes;
	availableLaps?: Array<number>;
	sessionId: string;
	currentLapId: number;
}

export default function TelemetryPage({
	initialTelemetryData,
	availableLaps,
	sessionId,
	currentLapId,
}: TelemetryPageProps) {
	const nav = useNavigate();
	const { data } = useSWR<TelemetryRes, Error>(
		`/api/sessions/${sessionId}/laps/2/geojson`,
		fetcher,
	);

	const [selectedMetric, setSelectedMetric] = useState<string>("Speed");
	const [_isScrubbing, setIsScrubbing] = useState<boolean>(false);

	const dataWithGPSCoordinates = useMemo(() => {
		return initialTelemetryData?.dataWithGPSCoordinates || [];
	}, [initialTelemetryData?.dataWithGPSCoordinates]);

	const { selectedIndex, handlePointSelection } = useTrackPosition(
		dataWithGPSCoordinates as TelemetryDataPoint[],
	);

	const trackInfo = useMemo(() => {
		if (dataWithGPSCoordinates.length === 0) return null;

		const firstPoint = dataWithGPSCoordinates[0] as TelemetryDataPoint;
		const lastPoint = dataWithGPSCoordinates[
			dataWithGPSCoordinates.length - 1
		] as TelemetryDataPoint;
		return {
			lapTime: lastPoint?.LapCurrentLapTime,
			trackName: firstPoint?.TrackName || "Unknown Track",
			sessionNum: firstPoint?.SessionNum || sessionId,
		};
	}, [dataWithGPSCoordinates, sessionId]);

	const hoverTimeoutRef = useRef<NodeJS.Timeout | null>(null);

	const handleChartHover = useCallback(() => {
		if (hoverTimeoutRef.current) {
			clearTimeout(hoverTimeoutRef.current);
		}
	}, []);

	const handleChartClick = useCallback(
		(index: number) => {
			// Clear any pending hover timeouts on click
			if (hoverTimeoutRef.current) {
				clearTimeout(hoverTimeoutRef.current);
			}

			handlePointSelection(index);
			setIsScrubbing(true);
			setTimeout(() => setIsScrubbing(false), 300);
		},
		[handlePointSelection],
	);

	const handleChartMouseLeave = useCallback(() => {
		// Clear any pending hover timeouts
		if (hoverTimeoutRef.current) {
			clearTimeout(hoverTimeoutRef.current);
		}
	}, []);

	// Cleanup timeouts on unmount
	useEffect(() => {
		return () => {
			if (hoverTimeoutRef.current) {
				clearTimeout(hoverTimeoutRef.current);
			}
		};
	}, []);

	// Memoize the telemetry data to prevent unnecessary recalculations
	const memoizedTelemetryData = useMemo(() => {
		return dataWithGPSCoordinates as TelemetryDataPoint[];
	}, [dataWithGPSCoordinates]);

	const handleLapChange = (newLapId: string) => {
		nav({ to: ".", search: () => ({ lapId: newLapId }) });
	};

	return (
		<div className="flex min-h-screen min-w-screen bg-zinc-950">
			{/* Sidebar */}
			<div className="flex w-64 flex-col border-zinc-800/50 border-r bg-zinc-900/50">
				{/* Logo/Brand */}
				<div className="px-6 py-6">
					<Link to="/" className="cursor-pointer">
						<div className="flex items-center space-x-3">
							<div className="flex h-8 w-8 items-center justify-center rounded-lg bg-white">
								<div className="h-4 w-4 rounded bg-zinc-900" />
							</div>
							<div>
								<h1 className="font-semibold text-sm text-white">iRacing</h1>
								<p className="text-xs text-zinc-400">Telemetry</p>
							</div>
						</div>
					</Link>
				</div>

				<nav className="flex-1 space-y-1 px-4">
					<div className="flex cursor-pointer items-center justify-between rounded-md px-3 py-2 font-medium text-sm text-white">
						Lap:
						<select
							className="h-fit rounded border border-zinc-600 bg-zinc-800/90 px-3 py-1 font-medium text-sm text-white hover:bg-zinc-700/90 focus:outline-none focus:ring-2 focus:ring-blue-500"
							onChange={(e) => handleLapChange(e.currentTarget.value)}
							value={(currentLapId || "")?.toString()}
						>
							{availableLaps?.map((lap) => (
								<option key={lap.toString()} value={lap.toString()}>
									{lap.toString()}
								</option>
							))}
						</select>
					</div>
					<div className="px-2 py-2 font-medium text-xs text-zinc-500 uppercase tracking-wider">
						Analysis
					</div>
					<div className="rounded-md bg-zinc-800/50 px-3 py-2 font-medium text-sm text-white">
						Session {sessionId}
					</div>
					<div className="cursor-pointer rounded-md px-3 py-2 font-medium text-sm text-zinc-400 hover:bg-zinc-800/50 hover:text-white">
						Track Map
					</div>
					<div className="cursor-pointer rounded-md px-3 py-2 font-medium text-sm text-zinc-400 hover:bg-zinc-800/50 hover:text-white">
						Telemetry
					</div>
					<div className="cursor-pointer rounded-md px-3 py-2 font-medium text-sm text-zinc-400 hover:bg-zinc-800/50 hover:text-white">
						Performance
					</div>
				</nav>
			</div>

			<div className="flex flex-1 flex-col">
				<main className="flex-1 space-y-6 p-6">
					{dataWithGPSCoordinates.length > 0 && (
						<div className="rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-4">
							<div className="grid grid-cols-2 gap-4 md:grid-cols-4">
								<div className="text-center">
									<div className="mb-1 text-xs text-zinc-500">Track</div>
									<div className="font-semibold text-lg text-white">
										{trackInfo?.trackName || "Unknown"}
									</div>
								</div>
								<div className="text-center">
									<div className="mb-1 text-xs text-zinc-500">GPS Points</div>
									<div className="font-semibold text-green-400 text-lg">
										{dataWithGPSCoordinates.length.toLocaleString()}
									</div>
								</div>
								<div className="text-center">
									<div className="mb-1 text-xs text-zinc-500">Max Speed</div>
									<div className="font-semibold text-lg text-yellow-400">
										{Math.max(
											...dataWithGPSCoordinates.map((p) => p.Speed || 0),
										).toFixed(0)}
										km/h
									</div>
								</div>
								<div className="text-center">
									<div className="mb-1 text-xs text-zinc-500">Lap time</div>
									<div className="font-semibold text-blue-400 text-lg">
										{trackInfo?.lapTime
											? formatTime(trackInfo?.lapTime)
											: "0.00"}
									</div>
								</div>
							</div>
						</div>
					)}

					<div className="grid grid-cols-1 gap-6 lg:grid-cols-5">
						<div className="col-span-1 rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-6 lg:col-span-3">
							<Card className="h-full w-full overflow-hidden p-0">
								{dataWithGPSCoordinates[0].Lon !== undefined && (
									<Map
										center={[
											dataWithGPSCoordinates[0].Lon,
											dataWithGPSCoordinates[dataWithGPSCoordinates.length / 2]
												.Lat,
										]}
										styles={{
											light: {
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
											},
										}}
										zoom={15}
									>
										<MapRoute
											coordinates={dataWithGPSCoordinates.map((data) => [
												data.Lon,
												data.Lat,
											])}
											color="#3b82f6"
											width={0.5}
											opacity={0}
										/>
										<RacingLine dataWithGPSCoordinates={data} />
										<MapControls
											showZoom
											showCompass
											showLocate
											showFullscreen
										/>
									</Map>
								)}
							</Card>
						</div>

						<div className="col-span-1 rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-4 lg:col-span-2">
							{memoizedTelemetryData.length > 0 ? (
								<ProfessionalTelemetryCharts
									telemetryData={memoizedTelemetryData}
									selectedIndex={selectedIndex}
									onHover={handleChartHover}
									onIndexChange={handleChartClick}
									onMouseLeave={handleChartMouseLeave}
								/>
							) : (
								<div className="flex h-150 items-center justify-center">
									<div className="text-center">
										<div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-lg bg-zinc-700/50">
											<div className="h-8 w-8 rounded border-2 border-zinc-600 border-dashed" />
										</div>
										<p className="mb-2 text-zinc-400">
											No telemetry data available
										</p>
										<p className="text-sm text-zinc-500">
											Loading telemetry charts...
										</p>
									</div>
								</div>
							)}
						</div>
					</div>

					{dataWithGPSCoordinates.length > 0 && (
						<InfoBox
							telemetryData={dataWithGPSCoordinates as TelemetryDataPoint[]}
							lapId={currentLapId.toString()}
							selectedMetric={selectedMetric}
							setSelectedMetric={setSelectedMetric}
						/>
					)}

					{dataWithGPSCoordinates.length > 0 && (
						<GPSAnalysisPanel data={dataWithGPSCoordinates} />
					)}
				</main>
			</div>
		</div>
	);
}

function RacingLine({
	dataWithGPSCoordinates,
}: {
	dataWithGPSCoordinates: any;
}) {
	const { map } = useMap();

	if (!map || !dataWithGPSCoordinates?.features) return;

	// Add source with the entire GeoJSON
	if (!map.getSource("racing-line")) {
		map.addSource("racing-line", {
			type: "geojson",
			data: dataWithGPSCoordinates,
		});
	}

	// Add layer with color from properties
	if (!map.getLayer("racing-line-layer")) {
		map.addLayer({
			id: "racing-line-layer",
			type: "line",
			source: "racing-line",
			paint: {
				"line-color": ["get", "color"], // ← Use color from properties
				"line-width": 4,
				"line-opacity": 1,
			},
			layout: {
				"line-cap": "round",
				"line-join": "round",
			},
		});
	}

	return <></>;
}
function GPSAnalysisPanel({ data }: { data: any[] }) {
	const totalDistance = data.reduce(
		(sum, point) => sum + (point.distanceFromPrev || 0),
		0,
	);
	const avgSpeed =
		data.reduce((sum, point) => sum + (point.Speed || 0), 0) / data.length;
	const maxSpeed = Math.max(...data.map((point) => point.Speed || 0));
	const minSpeed = Math.min(...data.map((point) => point.Speed || 0));

	const corners = data.filter((point) => point.sectionType === "corner");

	return (
		<div className="rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-6">
			<h2 className="mb-6 font-semibold text-lg text-white">
				GPS Track Analysis
			</h2>
			<div className="grid grid-cols-2 gap-4 md:grid-cols-4">
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/50 p-4">
					<div className="mb-2 text-sm text-zinc-400">Total Distance</div>
					<div className="font-bold text-2xl text-white">
						{(totalDistance / 1000).toFixed(2)} km
					</div>
				</div>
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/50 p-4">
					<div className="mb-2 text-sm text-zinc-400">Average Speed</div>
					<div className="font-bold text-2xl text-green-400">
						{avgSpeed.toFixed(1)} km/h
					</div>
				</div>
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/50 p-4">
					<div className="mb-2 text-sm text-zinc-400">Speed Range</div>
					<div className="font-bold text-2xl text-yellow-400">
						{minSpeed.toFixed(0)} - {maxSpeed.toFixed(0)}
					</div>
					<div className="text-xs text-zinc-500">km/h</div>
				</div>
				<div className="rounded-lg border border-zinc-700/50 bg-zinc-800/50 p-4">
					<div className="mb-2 text-sm text-zinc-400">Corner Points</div>
					<div className="font-bold text-2xl text-purple-400">
						{corners.length.toLocaleString()}
					</div>
					<div className="text-xs text-zinc-500">
						{((corners.length / data.length) * 100).toFixed(1)}% of lap
					</div>
				</div>
			</div>
		</div>
	);
}

function formatTime(totalSeconds: number | undefined) {
	if (!totalSeconds) return "--";
	const minutes = Math.floor(totalSeconds / 60);
	const remainingSeconds = totalSeconds % 60;

	// Extract milliseconds from the remaining seconds
	const seconds = Math.floor(remainingSeconds);
	const milliseconds = Math.round((remainingSeconds % 1) * 1000); // Round to nearest millisecond

	// Helper to pad single digits with a leading zero
	const padTo2Digits = (num: number) => {
		return num.toString().padStart(2, "0");
	};

	// Format the output string
	const paddedMinutes = padTo2Digits(minutes);
	const paddedSeconds = padTo2Digits(seconds);
	// Only show milliseconds if they exist
	const paddedMilliseconds = milliseconds > 0 ? `.${milliseconds}` : "";

	return `${paddedMinutes}:${paddedSeconds}${paddedMilliseconds}`;
}
