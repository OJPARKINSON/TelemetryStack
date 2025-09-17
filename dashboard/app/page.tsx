import Link from "next/link";
import ClientWrapper from "@/components/ClientWrapper";
import SessionSelector from "@/components/SessionSelector";
import { getSessions } from "@/lib/questDb";

export const dynamic = "force-dynamic";
export const revalidate = 0;

export default async function DashboardPage() {
	let sessions: string | any[] = [];
	let errorMessage = null;

	try {
		console.log("🏠 Dashboard page: Fetching sessions at RUNTIME...");
		sessions = await getSessions();
		console.log(`🏠 Dashboard page: Found ${sessions.length} sessions`);
	} catch (error) {
		console.error("🏠 Dashboard page: Error loading sessions:", error);
		errorMessage =
			error instanceof Error ? error.message : "Unknown error occurred";
	}

	return (
		<ClientWrapper>
			<div className="min-h-screen bg-zinc-950 flex">
				<div className="w-64 bg-zinc-900/50 border-r border-zinc-800/50 flex flex-col">
					<div className="px-6 py-6">
						<div className="flex items-center space-x-3">
							<div className="w-8 h-8 bg-white rounded-lg flex items-center justify-center">
								<div className="w-4 h-4 bg-zinc-900 rounded"></div>
							</div>
							<div>
								<Link href="/">
									<h1 className="text-sm font-semibold text-white">iRacing</h1>
									<p className="text-xs text-zinc-400">Telemetry</p>
								</Link>
							</div>
						</div>
					</div>

					<nav className="flex-1 px-4 space-y-1">
						<div className="px-2 py-2 text-xs font-medium text-zinc-500 uppercase tracking-wider">
							Dashboard
						</div>
						<div className="bg-zinc-800/50 text-white px-3 py-2 rounded-md text-sm font-medium">
							Sessions
						</div>
						<div className="text-zinc-400 hover:text-white hover:bg-zinc-800/50 px-3 py-2 rounded-md text-sm font-medium cursor-pointer">
							Analytics
						</div>
						<div className="text-zinc-400 hover:text-white hover:bg-zinc-800/50 px-3 py-2 rounded-md text-sm font-medium cursor-pointer">
							Settings
						</div>
					</nav>

					<div className="p-4 border-t border-zinc-800/50">
						<div className="flex items-center space-x-2">
							<div
								className={`h-2 w-2 rounded-full ${errorMessage ? "bg-red-400" : "bg-green-400"}`}
							></div>
							<span className="text-xs text-zinc-400">
								{errorMessage ? "Offline" : "Connected"}
							</span>
						</div>
					</div>
				</div>

				<div className="flex-1 flex flex-col">
					<header className="bg-zinc-950/50 border-b border-zinc-800/50 px-6 py-4">
						<div className="flex items-center space-x-2 text-sm">
							<span className="text-zinc-500">Dashboard</span>
							<span className="text-zinc-500">/</span>
							<span className="text-white">Sessions</span>
						</div>
					</header>

					<main className="flex-1 p-6">
						<div className="mb-8">
							<h1 className="text-2xl font-bold text-white mb-2">Sessions</h1>
							<p className="text-zinc-400">
								Manage and analyze your telemetry sessions
							</p>
						</div>

						<div className="space-y-6">
							{errorMessage ? (
								<div className="bg-red-950/50 border border-red-800/50 rounded-lg p-6">
									<div className="flex items-start space-x-3">
										<div className="flex-shrink-0">
											<div className="h-5 w-5 rounded-full bg-red-500/20 flex items-center justify-center">
												<div className="h-2 w-2 rounded-full bg-red-400"></div>
											</div>
										</div>
										<div className="flex-1">
											<h3 className="text-sm font-medium text-red-300">
												Database Connection Error
											</h3>
											<p className="text-sm text-red-200 mt-1">
												Unable to connect to QuestDB. Please check your
												configuration.
											</p>
											<details className="mt-3">
												<summary className="cursor-pointer text-xs text-red-300 hover:text-red-200">
													Show error details
												</summary>
												<div className="mt-2 p-3 bg-red-900/30 rounded border border-red-800/50">
													<code className="text-xs text-red-200 font-mono">
														{errorMessage}
													</code>
												</div>
											</details>
										</div>
									</div>
								</div>
							) : sessions.length > 0 ? (
								<SessionSelector sessions={sessions} />
							) : (
								<div className="bg-zinc-900/50 border border-zinc-800/50 rounded-lg p-12 text-center">
									<div className="w-16 h-16 mx-auto bg-zinc-800/50 rounded-lg flex items-center justify-center mb-4">
										<div className="w-8 h-8 border-2 border-zinc-600 rounded border-dashed"></div>
									</div>
									<h3 className="text-lg font-medium text-white mb-2">
										No sessions found
									</h3>
									<p className="text-zinc-400">
										Import telemetry data to get started with session analysis.
									</p>
								</div>
							)}

							{sessions.length > 0 && (
								<div className="grid grid-cols-1 md:grid-cols-3 gap-6">
									<div className="bg-zinc-900/50 border border-zinc-800/50 rounded-lg p-4">
										<div className="flex items-center justify-between mb-3">
											<h3 className="text-sm font-medium text-zinc-300">
												Database
											</h3>
											<div
												className={`h-2 w-2 rounded-full ${errorMessage ? "bg-red-400" : "bg-green-400"}`}
											></div>
										</div>
										<p className="text-lg font-semibold text-white mb-1">
											{errorMessage ? "Offline" : "Online"}
										</p>
										<p className="text-xs text-zinc-500">QuestDB Connection</p>
									</div>

									<div className="bg-zinc-900/50 border border-zinc-800/50 rounded-lg p-4">
										<div className="flex items-center justify-between mb-3">
											<h3 className="text-sm font-medium text-zinc-300">
												Sessions
											</h3>
											<div className="h-2 w-2 rounded-full bg-blue-400"></div>
										</div>
										<p className="text-lg font-semibold text-white mb-1">
											{sessions.length}
										</p>
										<p className="text-xs text-zinc-500">
											Available for analysis
										</p>
									</div>

									<div className="bg-zinc-900/50 border border-zinc-800/50 rounded-lg p-4">
										<div className="flex items-center justify-between mb-3">
											<h3 className="text-sm font-medium text-zinc-300">
												Processing
											</h3>
											<div className="h-2 w-2 rounded-full bg-green-400"></div>
										</div>
										<p className="text-lg font-semibold text-white mb-1">
											Active
										</p>
										<p className="text-xs text-zinc-500">Runtime dynamic</p>
									</div>
								</div>
							)}
						</div>
					</main>
				</div>
			</div>
		</ClientWrapper>
	);
}
