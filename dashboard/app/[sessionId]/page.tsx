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

	if (Number.isNaN(currentLapId)) {
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
		<div className="flex min-h-screen items-center justify-center bg-zinc-950">
			<div className="mx-auto max-w-md p-8 text-center">
				<div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-red-500/20">
					<div className="h-8 w-8 text-red-400">
						<svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<title>Error</title>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"
							/>
						</svg>
					</div>
				</div>
				<h1 className="mb-4 font-bold text-2xl text-white">
					Database Unavailable
				</h1>
				<p className="mb-6 text-zinc-400">
					The telemetry database is not running or accessible. Please start the
					Docker Compose stack to access telemetry data.
				</p>
				<div className="rounded-lg bg-zinc-800/50 p-4 text-left">
					<h3 className="mb-2 font-semibold text-sm text-zinc-300">
						To start the system:
					</h3>
					<code className="block rounded bg-zinc-900 px-2 py-1 text-xs text-zinc-400">
						docker compose up -d
					</code>
				</div>
				<div className="mt-6">
					<Link
						href="/dashboard"
						className="inline-flex items-center rounded-lg bg-blue-600 px-4 py-2 text-white transition-colors hover:bg-blue-700"
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
		<div className="flex min-h-screen bg-zinc-950">
			<div className="flex w-64 flex-col border-zinc-800/50 border-r bg-zinc-900/50">
				<div className="px-6 py-6">
					<div className="animate-pulse">
						<div className="flex items-center space-x-3">
							<div className="h-8 w-8 rounded-lg bg-zinc-700" />
							<div>
								<div className="mb-1 h-4 w-16 rounded bg-zinc-700" />
								<div className="h-3 w-12 rounded bg-zinc-700" />
							</div>
						</div>
					</div>
				</div>
			</div>

			<div className="flex flex-1 flex-col">
				<div className="border-zinc-800/50 border-b bg-zinc-950/50 px-6 py-4">
					<div className="h-4 w-48 animate-pulse rounded bg-zinc-700" />
				</div>
				<div className="flex-1 p-6">
					<div className="animate-pulse space-y-6">
						<div className="grid grid-cols-3 gap-6">
							<div className="h-32 rounded-lg bg-zinc-800/50" />
							<div className="h-32 rounded-lg bg-zinc-800/50" />
							<div className="h-32 rounded-lg bg-zinc-800/50" />
						</div>
						<div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
							<div className="col-span-2 h-96 rounded-lg bg-zinc-800/50" />
							<div className="h-96 rounded-lg bg-zinc-800/50" />
						</div>
					</div>
				</div>
			</div>
		</div>
	);
}
