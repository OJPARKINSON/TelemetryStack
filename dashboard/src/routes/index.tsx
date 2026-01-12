import { createFileRoute, Link } from "@tanstack/react-router";
import useSWR from "swr";
import SessionSelector, {
	type Session,
} from "../../components/SessionSelector";

// import { fetcher } from "@/lib/Fetch";

const fetcher = (url: string) =>
	fetch(url, {
		mode: "no-cors",
		method: "GET",
		headers: { "Content-Type": "application/json", "Content-Encoding": "gzip" },
	}).then((res) => {
		console.log(res);
		return res.json() as unknown as Session[];
	});

export const Route = createFileRoute("/")({
	component: RouteComponent,
});

function RouteComponent() {
	const {
		data: sessions,
		error: errorMessage,
		isLoading,
	} = useSWR<Session[], Error>("/api/sessions", fetcher);

	if (isLoading) return <div>Loading...</div>;
	return (
		<>
			<div className="flex w-64 flex-col border-zinc-800/50 border-r bg-zinc-900/50">
				<div className="px-6 py-6">
					<Link to="." className="cursor-pointer">
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
					<div className="px-2 py-2 font-medium text-xs text-zinc-500 uppercase tracking-wider">
						Dashboard
					</div>
					<div className="rounded-md bg-zinc-800/50 px-3 py-2 font-medium text-sm text-white">
						Sessions
					</div>
					<div className="cursor-pointer rounded-md px-3 py-2 font-medium text-sm text-zinc-400 hover:bg-zinc-800/50 hover:text-white">
						Analytics
					</div>
					<div className="cursor-pointer rounded-md px-3 py-2 font-medium text-sm text-zinc-400 hover:bg-zinc-800/50 hover:text-white">
						Settings
					</div>
				</nav>

				<div className="border-zinc-800/50 border-t p-4">
					<div className="flex items-center space-x-2">
						<div
							className={`h-2 w-2 rounded-full ${errorMessage ? "bg-red-400" : "bg-green-400"}`}
						/>
						<span className="text-xs text-zinc-400">
							{errorMessage ? "Offline" : "Connected"}
						</span>
					</div>
				</div>
			</div>
			<div className="flex flex-1 flex-col">
				<header className="border-zinc-800/50 border-b bg-zinc-950/50 px-6 py-4">
					<div className="flex items-center space-x-2 text-sm">
						<span className="text-zinc-500">Dashboard</span>
						<span className="text-zinc-500">/</span>
						<span className="text-white">Sessions</span>
					</div>
				</header>

				<main className="flex-1 p-6">
					<div className="mb-8">
						<h1 className="mb-2 font-bold text-2xl text-white">Sessions</h1>
						<p className="text-zinc-400">
							Manage and analyze your telemetry sessions
						</p>
					</div>

					<div className="space-y-6">
						{sessions === undefined ? (
							<div className="rounded-lg border border-red-800/50 bg-red-950/50 p-6">
								<div className="flex items-start space-x-3">
									<div className="shrink-0">
										<div className="flex h-5 w-5 items-center justify-center rounded-full bg-red-500/20">
											<div className="h-2 w-2 rounded-full bg-red-400" />
										</div>
									</div>
									<div className="flex-1">
										<h3 className="font-medium text-red-300 text-sm">
											Database Connection Error
										</h3>
										<p className="mt-1 text-red-200 text-sm">
											The telemetry database is not running. Start the Docker
											Compose stack to access telemetry data.
										</p>
										<div className="mt-4 rounded-lg bg-zinc-900/50 p-4">
											<h4 className="mb-2 font-semibold text-xs text-zinc-300">
												To start the system:
											</h4>
											<code className="mb-2 block rounded bg-zinc-800 px-2 py-1 text-xs text-zinc-400">
												docker compose up -d
											</code>
											<p className="text-xs text-zinc-500">
												This will start QuestDB, RabbitMQ, and all required
												services.
											</p>
										</div>
										<details className="mt-3">
											<summary className="cursor-pointer text-red-300 text-xs hover:text-red-200">
												Show technical error details
											</summary>
											<div className="mt-2 rounded border border-red-800/50 bg-red-900/30 p-3">
												<code className="font-mono text-red-200 text-xs">
													{errorMessage?.message}
												</code>
											</div>
										</details>
									</div>
								</div>
							</div>
						) : sessions.length > 0 ? (
							<SessionSelector sessions={sessions} />
						) : (
							<div className="rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-12 text-center">
								<div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-lg bg-zinc-800/50">
									<div className="h-8 w-8 rounded border-2 border-zinc-600 border-dashed" />
								</div>
								<h3 className="mb-2 font-medium text-lg text-white">
									No sessions found
								</h3>
								<p className="text-zinc-400">
									Import telemetry data to get started with session analysis.
								</p>
							</div>
						)}

						{sessions !== undefined && sessions.length > 0 && (
							<div className="grid grid-cols-1 gap-6 md:grid-cols-3">
								<div className="rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-4">
									<div className="mb-3 flex items-center justify-between">
										<h3 className="font-medium text-sm text-zinc-300">
											Database
										</h3>
										<div
											className={`h-2 w-2 rounded-full ${errorMessage ? "bg-red-400" : "bg-green-400"}`}
										/>
									</div>
									<p className="mb-1 font-semibold text-lg text-white">
										{errorMessage ? "Offline" : "Online"}
									</p>
									<p className="text-xs text-zinc-500">QuestDB Connection</p>
								</div>

								<div className="rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-4">
									<div className="mb-3 flex items-center justify-between">
										<h3 className="font-medium text-sm text-zinc-300">
											Sessions
										</h3>
										<div className="h-2 w-2 rounded-full bg-blue-400" />
									</div>
									<p className="mb-1 font-semibold text-lg text-white">
										{sessions.length}
									</p>
									<p className="text-xs text-zinc-500">
										Available for analysis
									</p>
								</div>

								<div className="rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-4">
									<div className="mb-3 flex items-center justify-between">
										<h3 className="font-medium text-sm text-zinc-300">
											Processing
										</h3>
										<div className="h-2 w-2 rounded-full bg-green-400" />
									</div>
									<p className="mb-1 font-semibold text-lg text-white">
										Active
									</p>
									<p className="text-xs text-zinc-500">Runtime dynamic</p>
								</div>
							</div>
						)}
					</div>
				</main>
			</div>
		</>
	);
}
