import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import ClientWrapper from "../../components/ClientWrapper";
import TelemetryPage from "../../components/TelemetryPage";
import { getLaps, getTelemetryData } from "../../lib/questDb";

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
			return <DatabaseUnavailableError />;
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

		// Check if it's a database connection error
		if (
			error instanceof Error &&
			(error.message.includes("table does not exist") ||
				error.message.includes("Connection refused") ||
				error.message.includes("connect ECONNREFUSED"))
		) {
			return <DatabaseUnavailableError />;
		}

		notFound();
	}
}

function DatabaseUnavailableError() {
	return (
		<div className="min-h-screen bg-zinc-950 flex items-center justify-center">
			<div className="text-center max-w-md mx-auto p-8">
				<div className="w-16 h-16 mx-auto bg-red-500/20 rounded-full flex items-center justify-center mb-6">
					<div className="w-8 h-8 text-red-400">
						<svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"
							/>
						</svg>
					</div>
				</div>
				<h1 className="text-2xl font-bold text-white mb-4">
					Database Unavailable
				</h1>
				<p className="text-zinc-400 mb-6">
					The telemetry database is not running or accessible. Please start the
					Docker Compose stack to access telemetry data.
				</p>
				<div className="bg-zinc-800/50 rounded-lg p-4 text-left">
					<h3 className="text-sm font-semibold text-zinc-300 mb-2">
						To start the system:
					</h3>
					<code className="block text-xs text-zinc-400 bg-zinc-900 rounded px-2 py-1">
						docker compose up -d
					</code>
				</div>
				<div className="mt-6">
					<Link
						href="/dashboard"
						className="inline-flex items-center px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
					>
						← Back to Dashboard
					</Link>
				</div>
			</div>
		</div>
	);
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
