import { createFileRoute, useNavigate } from "@tanstack/react-router";
import type { FeatureCollection } from "geojson";
import { useEffect, useMemo, useState } from "react";
import useSWR from "swr";

import { DatabaseUnavailableError } from "../../components/efecto/ErrorStates";
import { Header } from "../../components/efecto/Header";
import { StatBox } from "../../components/efecto/StatBox";
import TelemetrySection from "../../components/efecto/TelemetrySection";
import { TrackStatsFooter } from "../../components/efecto/TrackStatsFooter";
import { useChartHover } from "../../hooks/useChartHover";
import { useTelemetryData } from "../../hooks/useTelemetryData";
import {
	fetcher,
	fetcherBR,
	type TelemetryRes,
	telemetryFetcher,
} from "../../lib/Fetch";
import { formatLapTime } from "../../lib/format";
import type { TelemetryDataPoint, TrackInfo } from "../../lib/types";

export const Route = createFileRoute("/$sessionId")({
	component: SessionPage,
	validateSearch: (
		search: Record<string, unknown>,
	): {
		lapId: number;
	} => ({
		lapId: Number(search?.lapId) || 1,
	}),
});

export default function SessionPage() {
	const nav = useNavigate();
	const { sessionId } = Route.useParams();
	const { lapId } = Route.useSearch();
	const currentLapId = lapId || 1;

	const {
		data: telemetryData,
		error,
		isLoading,
		isValidating,
	} = useSWR<TelemetryRes, Error>(
		`/api/sessions/${sessionId}/laps/${currentLapId}`,
		telemetryFetcher,
		{ keepPreviousData: true },
	);

	const { data: availableLaps, error: lapsError } = useSWR<
		Array<number>,
		Error
	>(`/api/sessions/${sessionId}/laps`, fetcher);

	// Undefined while the lap list loads; 0 if it failed.
	const totalLaps = lapsError ? 0 : availableLaps?.length;

	const handleLapChange = (newLapId: number) => {
		nav({
			to: ".",
			search: (prev) => ({ ...prev, lapId: newLapId }),
			replace: true,
		});
	};

	if (error) return <DatabaseUnavailableError />;

	// First load renders the same layout with placeholders, so nothing moves
	// when the data arrives.
	return (
		<SessionPageInner
			loading={isLoading || !telemetryData}
			telemetryData={telemetryData}
			isValidating={isValidating}
			sessionId={sessionId}
			currentLapId={currentLapId}
			totalLaps={totalLaps}
			onLapChange={handleLapChange}
		/>
	);
}

interface SessionPageInnerProps {
	loading: boolean;
	telemetryData: TelemetryRes | undefined;
	isValidating: boolean;
	sessionId: string;
	currentLapId: number;
	totalLaps: number | undefined;
	onLapChange: (lapId: number) => void;
}

function SessionPageInner({
	loading,
	telemetryData,
	isValidating,
	sessionId,
	currentLapId,
	totalLaps,
	onLapChange,
}: SessionPageInnerProps) {
	const [showLoading, setShowLoading] = useState(false);
	useEffect(() => {
		if (!isValidating) {
			setShowLoading(false);
			return;
		}
		const id = setTimeout(() => setShowLoading(true), 150);
		return () => clearTimeout(id);
	}, [isValidating]);

	const { hoveredIndex, handleChartHover, handleChartMouseLeave } =
		useChartHover();
	const { dataWithGPSCoordinates, trackInfo, hoverCoordinates } =
		useTelemetryData(telemetryData, sessionId, hoveredIndex);

	const { data: racingLineData } = useSWR<FeatureCollection, Error>(
		`/api/sessions/${sessionId}/laps/${currentLapId}/geojson`,
		fetcherBR,
		{ keepPreviousData: true },
	);

	// 404 for laps ingested before the Tracks table existed.
	const { data: track } = useSWR<TrackInfo, Error>(
		`/api/sessions/${sessionId}/laps/${currentLapId}/track`,
		fetcherBR,
		{ keepPreviousData: true, shouldRetryOnError: false },
	);

	const data = dataWithGPSCoordinates as TelemetryDataPoint[];

	const { avgSpeed, lastPoint, fuelLapsEstimate } = useMemo(() => {
		if (data.length === 0)
			return { avgSpeed: 0, lastPoint: null, fuelLapsEstimate: 0 };

		const avg = data.reduce((sum, p) => sum + (p.Speed || 0), 0) / data.length;
		const last = data[data.length - 1];
		const first = data[0];
		const fuelUsed = first.FuelLevel - last.FuelLevel;
		const fuelEst = fuelUsed > 0 ? Math.floor(last.FuelLevel / fuelUsed) : 0;

		return { avgSpeed: avg, lastPoint: last, fuelLapsEstimate: fuelEst };
	}, [data]);

	return (
		<div className="flex flex-col bg-background border border-border w-full mx-auto min-h-screen">
			<Header
				trackName={
					loading ? undefined : trackInfo?.trackName || "Unknown Track"
				}
				sessionNum={sessionId}
				currentLapId={currentLapId}
				totalLaps={totalLaps}
				onLapChange={onLapChange}
				lapTime={loading ? undefined : formatLapTime(trackInfo?.lapTime)}
				onPrevLap={() => {
					if (currentLapId > 1) onLapChange(currentLapId - 1);
				}}
				onNextLap={() => {
					if (totalLaps && currentLapId < totalLaps)
						onLapChange(currentLapId + 1);
				}}
			/>
			<div className="flex items-stretch border-b border-border w-full bg-background">
				<StatBox
					title="Lap Time"
					loading={loading}
					stat={formatLapTime(trackInfo?.lapTime)}
					delta={
						trackInfo?.lapTime
							? `Δ${(trackInfo.lapTime % 1).toFixed(3).slice(1)}`
							: undefined
					}
				/>
				<StatBox
					title="Max Speed"
					loading={loading}
					stat={`${(trackInfo?.maxSpeed ?? 0).toFixed(1)}`}
					unit="km/h"
				/>
				<StatBox
					title="Position"
					loading={loading}
					stat={`P${lastPoint?.PlayerCarPosition || "-"}`}
					delta={
						lastPoint?.PlayerCarPosition && lastPoint.PlayerCarPosition > 1
							? `+${lastPoint.PlayerCarPosition - 1}`
							: undefined
					}
				/>
				<StatBox
					title="Avg Speed"
					loading={loading}
					stat={`${avgSpeed.toFixed(1)}`}
					unit="km/h"
				/>
				<StatBox
					title="Fuel Remaining"
					loading={loading}
					stat={`${(lastPoint?.FuelLevel ?? 0).toFixed(1)}`}
					unit="L"
					delta={fuelLapsEstimate > 0 ? `~${fuelLapsEstimate} laps` : undefined}
				/>
			</div>
			{/* Zero-height anchor: the progress bar overlays content instead of pushing it down */}
			<div className="relative h-0 w-full">
				{showLoading && (
					<div className="absolute inset-x-0 top-0 z-10 h-0.5 bg-border overflow-hidden">
						<div className="h-full w-1/3 bg-primary animate-slide" />
					</div>
				)}
			</div>
			<div className="flex flex-col flex-1">
				<TelemetrySection
					dataWithGPSCoordinates={data}
					racingLineData={racingLineData}
					sectorStartPct={track?.sector_start_pct}
					hoverCoordinates={hoverCoordinates}
					hoveredIndex={hoveredIndex}
					onHover={handleChartHover}
					onMouseLeave={handleChartMouseLeave}
					loading={loading}
				/>
			</div>
			<TrackStatsFooter data={data} loading={loading} />
		</div>
	);
}
