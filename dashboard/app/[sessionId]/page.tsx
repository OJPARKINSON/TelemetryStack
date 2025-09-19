import { notFound } from "next/navigation";
import { Suspense } from "react";
import { getLaps, getTelemetryData } from "../../lib/questDb";
import ClientWrapper from "../../components/ClientWrapper";
import TelemetryPage from "../../components/TelemetryPage";

// Force dynamic rendering - prevent build-time execution
export const dynamic = "force-dynamic";
export const revalidate = 0;

interface PageProps {
	params: Promise<{ sessionId: string }>;
	searchParams: Promise<{ lapId?: string }>;
}

export default async function SessionPage({ params, searchParams }: PageProps) {
	const { sessionId } = await params;
	const { lapId } = await searchParams;

	// Default to lap 1 if no lapId provided
	const currentLapId = lapId ? Number.parseInt(lapId, 10) : 1;

	if (isNaN(currentLapId)) {
		notFound();
	}

	// Fetch data on the server at RUNTIME (not build time)
	try {
		console.log(
			`📊 Fetching data at RUNTIME for session: ${sessionId}, lap: ${currentLapId}`,
		);

		const [telemetryData, availableLaps] = await Promise.all([
			getTelemetryData(sessionId, currentLapId),
			getLaps(sessionId),
		]);

		if (!telemetryData) {
			notFound();
		}

		console.log(
			`✅ Successfully fetched telemetry data with ${telemetryData.dataWithGPSCoordinates?.length || 0} points`,
		);

		return (
			<ClientWrapper fallback={<TelemetryLoadingSkeleton />}>
				<Suspense fallback={<TelemetryLoadingSkeleton />}>
					<TelemetryPage
						initialTelemetryData={telemetryData}
						availableLaps={availableLaps}
						sessionId={sessionId}
						currentLapId={currentLapId}
					/>
				</Suspense>
			</ClientWrapper>
		);
	} catch (error) {
		console.error("Error fetching telemetry data at runtime:", error);
		notFound();
	}
}

function TelemetryLoadingSkeleton() {
	return (
		<div className="min-h-screen bg-zinc-950 flex">
			<div className="w-64 bg-zinc-900/50 border-r border-zinc-800/50 flex flex-col">
				<div className="px-6 py-6">
					<div className="animate-pulse">
						<div className="flex items-center space-x-3">
							<div className="w-8 h-8 bg-zinc-700 rounded-lg"></div>
							<div>
								<div className="h-4 w-16 bg-zinc-700 rounded mb-1"></div>
								<div className="h-3 w-12 bg-zinc-700 rounded"></div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<div className="flex-1 flex flex-col">
				<div className="bg-zinc-950/50 border-b border-zinc-800/50 px-6 py-4">
					<div className="animate-pulse h-4 w-48 bg-zinc-700 rounded"></div>
				</div>
				<div className="flex-1 p-6">
					<div className="animate-pulse space-y-6">
						<div className="grid grid-cols-3 gap-6">
							<div className="bg-zinc-800/50 h-32 rounded-lg"></div>
							<div className="bg-zinc-800/50 h-32 rounded-lg"></div>
							<div className="bg-zinc-800/50 h-32 rounded-lg"></div>
						</div>
						<div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
							<div className="col-span-2 bg-zinc-800/50 h-96 rounded-lg"></div>
							<div className="bg-zinc-800/50 h-96 rounded-lg"></div>
						</div>
					</div>
				</div>
			</div>
		</div>
	);
}
