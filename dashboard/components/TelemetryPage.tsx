import { useNavigate } from "@tanstack/react-router";
import React, { useState } from "react";
import useSWR from "swr";
import { InfoBox } from "../components/InfoBox";
import { fetcher, type TelemetryRes } from "../lib/Fetch";
import type { TelemetryDataPoint } from "../lib/types";
import { useChartHover } from "../hooks/useChartHover";
import { useTelemetryData } from "../hooks/useTelemetryData";
import {
	GPSAnalysisPanel,
	TelemetryMapSection,
	TelemetrySidebar,
	TelemetryStatsHeader,
} from "./telemetry";

const ProfessionalTelemetryCharts = React.lazy(
	() => import("./TelemetryCharts"),
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
	const [selectedMetric, setSelectedMetric] = useState<string>("Speed");
	const { hoveredIndex, handleChartHover, handleChartMouseLeave } =
		useChartHover();
	const { dataWithGPSCoordinates, trackInfo, hoverCoordinates } =
		useTelemetryData(initialTelemetryData, sessionId, hoveredIndex);

	const { data: racingLineData } = useSWR<GeoJSON.FeatureCollection, Error>(
		`/api/sessions/${sessionId}/laps/${currentLapId}/geojson`,
		fetcher,
	);

	const handleLapChange = (newLapId: string) => {
		nav({ to: ".", search: () => ({ lapId: newLapId }) });
	};

	return (
		<div className="flex min-h-screen min-w-screen bg-zinc-950">
			<TelemetrySidebar
				availableLaps={availableLaps}
				currentLapId={currentLapId}
				sessionId={sessionId}
				onLapChange={handleLapChange}
			/>

			<div className="flex flex-1 flex-col">
				<main className="flex-1 space-y-6 p-6 pt-0">
					{dataWithGPSCoordinates.length > 0 && (
						<TelemetryStatsHeader
							trackName={trackInfo?.trackName || "Unknown"}
							gpsPointCount={dataWithGPSCoordinates.length}
							maxSpeed={trackInfo?.maxSpeed || 0}
							lapTime={trackInfo?.lapTime}
						/>
					)}

					<div className="grid grid-cols-1 gap-6 lg:grid-cols-5">
						<TelemetryMapSection
							dataWithGPSCoordinates={dataWithGPSCoordinates}
							racingLineData={racingLineData}
							hoverCoordinates={hoverCoordinates}
						/>

						<div className="col-span-1 rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-4 lg:col-span-2">
							{dataWithGPSCoordinates.length > 0 ? (
								<ProfessionalTelemetryCharts
									telemetryData={
										dataWithGPSCoordinates as TelemetryDataPoint[]
									}
									onMouseLeave={handleChartMouseLeave}
									onHover={handleChartHover}
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
