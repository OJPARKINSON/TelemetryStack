"use client";

import dynamic from "next/dynamic";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import {
	useCallback,
	useDeferredValue,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import { InfoBox } from "../components/InfoBox";
import { useTrackPosition } from "../hooks/useTrackPosition";
import type { TelemetryRes } from "../lib/Fetch";
import type { TelemetryDataPoint } from "../lib/types";

const OptimizedTrackMap = dynamic(() => import("./OptimizedTrackMap"), {
	ssr: false,
	loading: () => <TrackMapSkeleton />,
});

const ProfessionalTelemetryCharts = dynamic(
	() => import("./ProfessionalTelemetryCharts"),
	{
		ssr: false,
		loading: () => <ChartsSkeleton />,
	},
);

interface TelemetryPageProps {
	initialTelemetryData: TelemetryRes;
	availableLaps: Array<{ lap_id: number }>;
	sessionId: string;
	currentLapId: number;
}

export default function TelemetryPage({
	initialTelemetryData,
	availableLaps,
	sessionId,
	currentLapId,
}: TelemetryPageProps) {
	const router = useRouter();
	const pathname = usePathname();
	const searchParams = useSearchParams();

	const [selectedMetric, setSelectedMetric] = useState<string>("Speed");
	const [isScrubbing, setIsScrubbing] = useState<boolean>(false);
	const [hoverIndex, setHoverIndex] = useState<number>(-1);

	// Extract processed data from the server response - wrap in useMemo to fix dependency warning
	const dataWithGPSCoordinates = useMemo(() => {
		return initialTelemetryData?.dataWithGPSCoordinates || [];
	}, [initialTelemetryData?.dataWithGPSCoordinates]);

	const processError = initialTelemetryData?.processError || null;

	const { selectedIndex, handlePointSelection } = useTrackPosition(
		dataWithGPSCoordinates as TelemetryDataPoint[],
	);

	// Derive track information from data
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

	// Memoize track point click handler
	const handleTrackPointClick = useCallback(
		(index: number) => {
			handlePointSelection(index);
			setIsScrubbing(true);
			setTimeout(() => setIsScrubbing(false), 500);
		},
		[handlePointSelection],
	);

	// Debounced hover handling for smoother interactions
	const hoverTimeoutRef = useRef<NodeJS.Timeout | null>(null);

	const handleChartHover = useCallback((index: number) => {
		// Clear any pending hover timeout
		if (hoverTimeoutRef.current) {
			clearTimeout(hoverTimeoutRef.current);
		}

		// More aggressive debouncing for heavy datasets
		hoverTimeoutRef.current = setTimeout(() => {
			setHoverIndex(index);
		}, 16); // 16ms debounce (~60fps) for better performance with 39K points
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
			setHoverIndex(-1); // Clear hover state after selection
		},
		[handlePointSelection],
	);

	const handleChartMouseLeave = useCallback(() => {
		// Clear any pending hover timeouts
		if (hoverTimeoutRef.current) {
			clearTimeout(hoverTimeoutRef.current);
		}

		// Debounce mouse leave to prevent flickering
		hoverTimeoutRef.current = setTimeout(() => {
			setHoverIndex(-1);
		}, 50); // Slightly longer delay for mouse leave
	}, []);

	// Cleanup timeouts on unmount
	useEffect(() => {
		return () => {
			if (hoverTimeoutRef.current) {
				clearTimeout(hoverTimeoutRef.current);
			}
		};
	}, []);

	// Defer hover index updates to prevent blocking the main thread
	const deferredHoverIndex = useDeferredValue(hoverIndex);

	// Get display index (hover takes precedence over selection for preview)
	const displayIndex = useMemo(() => {
		return deferredHoverIndex >= 0 ? deferredHoverIndex : selectedIndex;
	}, [deferredHoverIndex, selectedIndex]);

	// Memoize the telemetry data to prevent unnecessary recalculations
	const memoizedTelemetryData = useMemo(() => {
		return dataWithGPSCoordinates as TelemetryDataPoint[];
	}, [dataWithGPSCoordinates]);

	const handleLapChange = (newLapId: string) => {
		const params = new URLSearchParams();
		params.set("lapId", newLapId);
		router.push(`${pathname}?${params.toString()}`);
	};

	if (processError) {
		return (
			<div className="min-h-screen bg-zinc-950 flex">
				{/* Sidebar */}
				<div className="w-64 bg-zinc-900/50 border-r border-zinc-800/50 flex flex-col">
					{/* Logo/Brand */}
					<div className="px-6 py-6">
						<div className="flex items-center space-x-3">
							<div className="w-8 h-8 bg-white rounded-lg flex items-center justify-center">
								<div className="w-4 h-4 bg-zinc-900 rounded"></div>
							</div>
							<div>
								<h1 className="text-sm font-semibold text-white">iRacing</h1>
								<p className="text-xs text-zinc-400">Telemetry</p>
							</div>
						</div>
					</div>
				</div>

				{/* Main Content */}
				<div className="flex-1 flex flex-col">
					{/* Header */}
					<header className="bg-zinc-950/50 border-b border-zinc-800/50 px-6 py-4">
						<div className="flex items-center space-x-2 text-sm">
							<span className="text-zinc-500">Dashboard</span>
							<span className="text-zinc-500">/</span>
							<span className="text-zinc-500">Sessions</span>
						</div>
					</header>

					{/* Error Content */}
					<main className="flex-1 p-6 flex items-center justify-center">
						<div className="bg-red-950/50 border border-red-800/50 rounded-lg p-6 max-w-md text-center">
							<div className="w-16 h-16 mx-auto bg-red-900/50 rounded-lg flex items-center justify-center mb-4">
								<div className="w-8 h-8 border-2 border-red-400 rounded border-dashed"></div>
							</div>
							<h3 className="text-lg font-medium text-red-300 mb-2">
								Error Loading Telemetry Data
							</h3>
							<p className="text-red-200 text-sm">{processError}</p>
						</div>
					</main>
				</div>
			</div>
		);
	}

	return (
		<div className="min-h-screen bg-zinc-950 flex">
			{/* Sidebar */}
			<div className="w-64 bg-zinc-900/50 border-r border-zinc-800/50 flex flex-col">
				{/* Logo/Brand */}
				<div className="px-6 py-6">
					<Link href="/" className=" cursor-pointer">
						<div className="flex items-center space-x-3">
							<div className="w-8 h-8 bg-white rounded-lg flex items-center justify-center">
								<div className="w-4 h-4 bg-zinc-900 rounded"></div>
							</div>
							<div>
								<h1 className="text-sm font-semibold text-white">iRacing</h1>
								<p className="text-xs text-zinc-400">Telemetry</p>
							</div>
						</div>
					</Link>
				</div>

				<nav className="flex-1 px-4 space-y-1">
					<div className="text-white px-3 py-2 rounded-md text-sm font-medium cursor-pointer flex justify-between items-center">
						Lap:
						<select
							className="bg-zinc-800/90 border border-zinc-600 text-white px-3 py-1 rounded text-sm font-medium hover:bg-zinc-700/90 focus:outline-none focus:ring-2 focus:ring-blue-500 h-fit"
							onChange={(e) => handleLapChange(e.currentTarget.value)}
							value={(searchParams.get("lapId") || "")?.toString()}
						>
							{availableLaps.map(({ lap_id }) => (
								<option key={lap_id.toString()} value={lap_id.toString()}>
									{lap_id.toString()}
								</option>
							))}
						</select>
					</div>
					<div className="px-2 py-2 text-xs font-medium text-zinc-500 uppercase tracking-wider">
						Analysis
					</div>
					<div className="bg-zinc-800/50 text-white px-3 py-2 rounded-md text-sm font-medium">
						Session {sessionId}
					</div>
					<div className="text-zinc-400 hover:text-white hover:bg-zinc-800/50 px-3 py-2 rounded-md text-sm font-medium cursor-pointer">
						Track Map
					</div>
					<div className="text-zinc-400 hover:text-white hover:bg-zinc-800/50 px-3 py-2 rounded-md text-sm font-medium cursor-pointer">
						Telemetry
					</div>
					<div className="text-zinc-400 hover:text-white hover:bg-zinc-800/50 px-3 py-2 rounded-md text-sm font-medium cursor-pointer">
						Performance
					</div>
				</nav>
			</div>

			<div className="flex-1 flex flex-col">
				<main className="flex-1 p-6 space-y-6">
					{dataWithGPSCoordinates.length > 0 && (
						<div className="bg-zinc-900/50 border border-zinc-800/50 rounded-lg p-4">
							<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
								<div className="text-center">
									<div className="text-xs text-zinc-500 mb-1">Track</div>
									<div className="text-lg font-semibold text-white">
										{trackInfo?.trackName || "Unknown"}
									</div>
								</div>
								<div className="text-center">
									<div className="text-xs text-zinc-500 mb-1">GPS Points</div>
									<div className="text-lg font-semibold text-green-400">
										{dataWithGPSCoordinates.length.toLocaleString()}
									</div>
								</div>
								<div className="text-center">
									<div className="text-xs text-zinc-500 mb-1">Max Speed</div>
									<div className="text-lg font-semibold text-yellow-400">
										{Math.max(
											...dataWithGPSCoordinates.map((p) => p.Speed || 0),
										).toFixed(0)}{" "}
										km/h
									</div>
								</div>
								<div className="text-center">
									<div className="text-xs text-zinc-500 mb-1">Lap time</div>
									<div className="text-lg font-semibold text-blue-400">
										{trackInfo?.lapTime
											? formatTime(trackInfo?.lapTime)
											: "0.00"}
									</div>
								</div>
							</div>
						</div>
					)}

					<div className="grid grid-cols-1 lg:grid-cols-5 gap-6">
						<div className="col-span-1 lg:col-span-3 bg-zinc-900/50 border border-zinc-800/50 rounded-lg p-6">
							{dataWithGPSCoordinates.length > 0 ? (
								<OptimizedTrackMap
									dataWithCoordinates={dataWithGPSCoordinates}
									selectedPointIndex={displayIndex}
									onPointClick={handleTrackPointClick}
									selectedMetric={selectedMetric}
									setSelectedMetric={setSelectedMetric}
								/>
							) : (
								<div className="h-[500px] bg-zinc-800/50 rounded-lg flex items-center justify-center">
									<div className="text-center">
										<div className="w-16 h-16 mx-auto bg-zinc-700/50 rounded-lg flex items-center justify-center mb-4">
											<div className="w-8 h-8 border-2 border-zinc-600 rounded border-dashed"></div>
										</div>
										<p className="text-zinc-400 mb-2">No GPS data available</p>
										<p className="text-zinc-500 text-sm">
											This session may not contain GPS coordinates or they may
											be invalid.
										</p>
									</div>
								</div>
							)}
						</div>

						<div className="col-span-1 lg:col-span-2 bg-zinc-900/50 border border-zinc-800/50 rounded-lg p-4">
							{memoizedTelemetryData.length > 0 ? (
								<ProfessionalTelemetryCharts
									telemetryData={memoizedTelemetryData}
									selectedIndex={selectedIndex}
									onHover={handleChartHover}
									onIndexChange={handleChartClick}
									onMouseLeave={handleChartMouseLeave}
								/>
							) : (
								<div className="h-[600px] flex items-center justify-center">
									<div className="text-center">
										<div className="w-16 h-16 mx-auto bg-zinc-700/50 rounded-lg flex items-center justify-center mb-4">
											<div className="w-8 h-8 border-2 border-zinc-600 rounded border-dashed"></div>
										</div>
										<p className="text-zinc-400 mb-2">
											No telemetry data available
										</p>
										<p className="text-zinc-500 text-sm">
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
		<div className="bg-zinc-900/50 border border-zinc-800/50 rounded-lg p-6">
			<h2 className="text-lg font-semibold text-white mb-6">
				GPS Track Analysis
			</h2>
			<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
				<div className="bg-zinc-800/50 border border-zinc-700/50 rounded-lg p-4">
					<div className="text-sm text-zinc-400 mb-2">Total Distance</div>
					<div className="text-2xl font-bold text-white">
						{(totalDistance / 1000).toFixed(2)} km
					</div>
				</div>
				<div className="bg-zinc-800/50 border border-zinc-700/50 rounded-lg p-4">
					<div className="text-sm text-zinc-400 mb-2">Average Speed</div>
					<div className="text-2xl font-bold text-green-400">
						{avgSpeed.toFixed(1)} km/h
					</div>
				</div>
				<div className="bg-zinc-800/50 border border-zinc-700/50 rounded-lg p-4">
					<div className="text-sm text-zinc-400 mb-2">Speed Range</div>
					<div className="text-2xl font-bold text-yellow-400">
						{minSpeed.toFixed(0)} - {maxSpeed.toFixed(0)}
					</div>
					<div className="text-xs text-zinc-500">km/h</div>
				</div>
				<div className="bg-zinc-800/50 border border-zinc-700/50 rounded-lg p-4">
					<div className="text-sm text-zinc-400 mb-2">Corner Points</div>
					<div className="text-2xl font-bold text-purple-400">
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

// Performance optimization: Skeleton components for dynamic loading
function TrackMapSkeleton() {
	return (
		<div className="h-[700px] bg-zinc-800/30 rounded-lg flex items-center justify-center animate-pulse">
			<div className="text-center">
				<div className="w-16 h-16 mx-auto bg-zinc-700/50 rounded-lg flex items-center justify-center mb-4">
					<div className="w-8 h-8 border-2 border-zinc-600 rounded border-dashed animate-spin"></div>
				</div>
				<p className="text-zinc-400 text-sm">Loading track map...</p>
			</div>
		</div>
	);
}

function ChartsSkeleton() {
	return (
		<div className="h-[400px] bg-zinc-800/30 rounded-lg p-4 animate-pulse">
			<div className="space-y-4">
				<div className="h-4 bg-zinc-700/50 rounded w-1/3"></div>
				<div className="space-y-2">
					{[...Array(5)].map((_, i) => (
						// biome-ignore lint/suspicious/noArrayIndexKey: na
						<div key={i} className="h-16 bg-zinc-700/30 rounded"></div>
					))}
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
