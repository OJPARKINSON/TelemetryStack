import { createFileRoute, useNavigate } from "@tanstack/react-router";
import type { FeatureCollection } from "geojson";
import { useEffect, useMemo, useState } from "react";
import useSWR from "swr";

import { DatabaseUnavailableError } from "../../components/efecto/ErrorStates";
import { Header } from "../../components/efecto/Header";
import { NewTelemetrySkeleton } from "../../components/efecto/Skeleton";
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
import type { TelemetryDataPoint } from "../../lib/types";

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

	const { data: availableLaps } = useSWR<Array<number>, Error>(
		`/api/sessions/${sessionId}/laps`,
		fetcher,
	);

	const totalLaps = availableLaps?.length ?? 0;

	const handleLapChange = (newLapId: number) => {
		nav({
			to: ".",
			search: (prev) => ({ ...prev, lapId: newLapId }),
			replace: true,
		});
	};

	if (error) return <DatabaseUnavailableError />;

	// First load only — show skeleton
	if (isLoading || !telemetryData) return <NewTelemetrySkeleton />;

	return (
		<SessionPageInner
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
	telemetryData: TelemetryRes;
	isValidating: boolean;
	sessionId: string;
	currentLapId: number;
	totalLaps: number;
	onLapChange: (lapId: number) => void;
}

function SessionPageInner({
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
				trackName={trackInfo?.trackName || "Unknown Track"}
				sessionNum={sessionId}
				currentLapId={currentLapId}
				totalLaps={totalLaps}
				lapTime={formatLapTime(trackInfo?.lapTime)}
				onPrevLap={() => {
					if (currentLapId > 1) onLapChange(currentLapId - 1);
				}}
				onNextLap={() => {
					if (currentLapId < totalLaps) onLapChange(currentLapId + 1);
				}}
			/>
			<div className="flex items-stretch border-b border-border w-full bg-background">
				<StatBox
					title="Lap Time"
					stat={formatLapTime(trackInfo?.lapTime)}
					delta={
						trackInfo?.lapTime
							? `Δ${(trackInfo.lapTime % 1).toFixed(3).slice(1)}`
							: undefined
					}
				/>
				<StatBox
					title="Max Speed"
					stat={`${(trackInfo?.maxSpeed ?? 0).toFixed(1)}`}
					unit="km/h"
				/>
				<StatBox
					title="Position"
					stat={`P${lastPoint?.PlayerCarPosition || "-"}`}
					delta={
						lastPoint?.PlayerCarPosition && lastPoint.PlayerCarPosition > 1
							? `+${lastPoint.PlayerCarPosition - 1}`
							: undefined
					}
				/>
				<StatBox
					title="Avg Speed"
					stat={`${avgSpeed.toFixed(1)}`}
					unit="km/h"
				/>
				<StatBox
					title="Fuel Remaining"
					stat={`${(lastPoint?.FuelLevel ?? 0).toFixed(1)}`}
					unit="L"
					delta={fuelLapsEstimate > 0 ? `~${fuelLapsEstimate} laps` : undefined}
				/>
			</div>
			{showLoading && (
				<div className="h-0.5 w-full bg-border overflow-hidden">
					<div className="h-full w-1/3 bg-primary animate-slide" />
				</div>
			)}
			<div className="flex flex-col flex-1">
				<TelemetrySection
					dataWithGPSCoordinates={data}
					racingLineData={racingLineData}
					hoverCoordinates={hoverCoordinates}
					hoveredIndex={hoveredIndex}
					onHover={handleChartHover}
					onMouseLeave={handleChartMouseLeave}
				/>
			</div>
			<TrackStatsFooter data={data} />
		</div>
	);
}
