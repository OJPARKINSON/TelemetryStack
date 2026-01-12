import { createFileRoute, Link } from "@tanstack/react-router";
import { notFound } from "next/navigation";
import useSWR from "swr";
import TelemetryPage from "../../components/TelemetryPage";
import { processIRacingDataWithGPS, type TelemetryRes } from "../../lib/Fetch";

export const Route = createFileRoute("/$sessionId")({
	component: SessionPage,
	validateSearch: (
		search: Record<string, unknown>,
	): {
		lapId: string;
	} => ({
		lapId: (search?.lapId as string) || "1",
	}),
});

const fetcher = (url: string) =>
	fetch(url, {
		mode: "no-cors",
		method: "GET",
		headers: { "Content-Type": "application/json", "Content-Encoding": "gzip" },
	}).then(async (res) => {
		console.log(res);
		return processIRacingDataWithGPS(await res.json());
	});

const lapsFetcher = (url: string) =>
	fetch(url, {
		mode: "no-cors",
		method: "GET",
		headers: { "Content-Type": "application/json", "Content-Encoding": "gzip" },
	}).then((res) => {
		console.log(res);
		return res.json() as unknown as Array<number>;
	});

export default function SessionPage() {
	const { sessionId } = Route.useParams();
	const { lapId } = Route.useSearch();

	const {
		data: telemetryData,
		error,
		isLoading,
	} = useSWR<TelemetryRes, Error>(
		`/api/sessions/${sessionId}/laps/${lapId}`,
		fetcher,
	);

	const {
		data: availableLaps,
		error: error2,
		isLoading: isLoading2,
	} = useSWR<Array<number>, Error>(
		`/api/sessions/${sessionId}/laps`,
		lapsFetcher,
	);

	// Default to lap 1 if no lapId provided
	const currentLapId = lapId ? Number.parseInt(lapId, 10) : 1;

	if (Number.isNaN(currentLapId)) {
		notFound();
	}

	if (error) return <DatabaseUnavailableError />;
	if (isLoading || telemetryData === undefined)
		return <TelemetryLoadingSkeleton />;

	return (
		<TelemetryPage
			initialTelemetryData={telemetryData}
			availableLaps={availableLaps}
			sessionId={sessionId}
			currentLapId={currentLapId}
		/>
	);
}

function DatabaseUnavailableError() {
	return (
		<div>
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
						to=".."
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
