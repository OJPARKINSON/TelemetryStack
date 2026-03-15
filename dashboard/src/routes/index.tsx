import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";
import useSWR from "swr";
import SessionSelector, {
	type Session,
} from "../../components/SessionSelector";
import { StatBox } from "../../components/efecto/StatBox";
import { fetcher } from "../../lib/Fetch";

function RouteComponent() {
	const { data, error, isLoading } = useSWR<Session[], Error>(
		"/api/sessions",
		fetcher,
	);
	const [activeTrack, setActiveTrack] = useState<string | null>(null);

	if (isLoading) {
		return (
			<div className="flex flex-col bg-background w-full min-h-screen">
				<div className="flex items-center justify-between px-8 py-4 border-b border-border bg-background">
					<div className="flex items-center gap-2.5">
						<div className="w-7 h-7 rounded-md bg-primary flex items-center justify-center">
							<span className="text-primary-foreground text-xs font-bold">
								iR
							</span>
						</div>
						<span className="text-foreground text-sm font-medium tracking-tight">
							Telemetry
						</span>
						<span className="text-muted-foreground text-sm">/</span>
						<span className="text-muted-foreground text-sm">Sessions</span>
					</div>
				</div>
				<div className="flex-1 flex items-center justify-center">
					<div className="flex items-center gap-3">
						<div className="w-4 h-4 rounded-full border-2 border-primary border-t-transparent animate-spin" />
						<span className="text-muted-foreground text-sm">
							Loading sessions...
						</span>
					</div>
				</div>
			</div>
		);
	}

	const sessions = data ?? [];
	const sessionsByTrack = sessions.reduce(
		(acc, session) => {
			const track = session.track_name || "Unknown";
			if (!acc[track]) acc[track] = [];
			acc[track].push(session);
			return acc;
		},
		{} as Record<string, Session[]>,
	);

	const filteredSessions = activeTrack
		? sessions.filter((s) => (s.track_name || "Unknown") === activeTrack)
		: sessions;

	const totalLaps = sessions.reduce(
		(sum, s) => sum + Number(s.max_lap_id || 0),
		0,
	);

	const trackNames = Object.keys(sessionsByTrack);

	return (
		<div className="flex flex-col bg-background border border-border w-full mx-auto min-h-screen">
			{/* Header — matches session page Header component */}
			<div className="flex items-center justify-between px-8 py-4 border-b border-border w-full bg-background">
				<div className="flex items-center gap-6">
					<Link to="/">
						<div className="flex items-center gap-2.5">
							<div className="w-7 h-7 rounded-md bg-primary flex items-center justify-center">
								<span className="text-primary-foreground text-xs font-bold">
									iR
								</span>
							</div>
							<span className="text-foreground text-sm font-medium tracking-tight">
								Telemetry
							</span>
							<span className="text-muted-foreground text-sm">/</span>
							<span className="text-muted-foreground text-sm">Sessions</span>
						</div>
					</Link>
					{/* Track filter tabs */}
					{sessions.length > 0 && trackNames.length > 1 && (
						<div className="flex items-center gap-1 ml-4 max-w-xl overflow-x-auto">
							<button
								type="button"
								onClick={() => setActiveTrack(null)}
								className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors whitespace-nowrap ${
									activeTrack === null
										? "bg-secondary text-foreground"
										: "text-muted-foreground hover:text-foreground"
								}`}
							>
								All
							</button>
							{trackNames.map((trackName) => (
								<button
									type="button"
									key={trackName}
									onClick={() =>
										setActiveTrack(
											activeTrack === trackName ? null : trackName,
										)
									}
									className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors whitespace-nowrap ${
										activeTrack === trackName
											? "bg-secondary text-foreground"
											: "text-muted-foreground hover:text-foreground"
									}`}
								>
									{trackName}
								</button>
							))}
						</div>
					)}
				</div>
				<div className="flex items-center gap-2">
					<div
						className={`w-1.5 h-1.5 rounded-full ${error ? "bg-red-500" : "bg-emerald-500"}`}
					/>
					<span className="text-muted-foreground text-xs">
						{error ? "Offline" : "Connected"}
					</span>
				</div>
			</div>

			{/* Stats row — matches session page StatBox pattern */}
			{sessions.length > 0 && (
				<div className="flex items-stretch border-b border-border w-full bg-background">
					<StatBox
						title="Total Sessions"
						stat={String(sessions.length)}
					/>
					<StatBox
						title="Total Laps"
						stat={totalLaps.toLocaleString()}
					/>
					<StatBox
						title="Tracks"
						stat={String(trackNames.length)}
					/>
					<StatBox
						title="Database"
						stat={error ? "Offline" : "Online"}
					/>
				</div>
			)}

			{/* Content */}
			<div className="flex-1 overflow-y-auto">
				{data === undefined ? (
					<div className="p-8">
						<div className="rounded-lg border border-red-800/50 bg-red-950/50 p-6">
							<div className="flex items-start gap-3">
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
									<div className="mt-4 rounded-lg bg-card p-4">
										<h4 className="mb-2 font-semibold text-xs text-foreground">
											To start the system:
										</h4>
										<code className="mb-2 block rounded bg-secondary px-2 py-1 text-xs text-muted-foreground">
											docker compose up -d
										</code>
									</div>
									<details className="mt-3">
										<summary className="cursor-pointer text-red-300 text-xs hover:text-red-200">
											Show technical error details
										</summary>
										<div className="mt-2 rounded border border-red-800/50 bg-red-900/30 p-3">
											<code className="font-mono text-red-200 text-xs">
												{error?.message}
											</code>
										</div>
									</details>
								</div>
							</div>
						</div>
					</div>
				) : sessions.length > 0 ? (
					<SessionSelector sessions={filteredSessions} />
				) : (
					<div className="flex-1 flex items-center justify-center p-12">
						<div className="text-center">
							<div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-lg bg-secondary">
								<div className="h-8 w-8 rounded border-2 border-muted-foreground/30 border-dashed" />
							</div>
							<h3 className="mb-2 font-medium text-lg text-foreground">
								No sessions found
							</h3>
							<p className="text-muted-foreground text-sm">
								Import telemetry data to get started with session analysis.
							</p>
						</div>
					</div>
				)}
			</div>

			{/* Footer — matches session page TrackStatsFooter pattern */}
			{sessions.length > 0 && (
				<div className="flex items-stretch border-t border-border bg-background">
					{trackNames.map((trackName) => (
						<div
							key={trackName}
							className="flex-1 px-6 py-3 border-r border-border last:border-r-0"
						>
							<p className="text-muted-foreground text-xs mb-0.5">
								{trackName}
							</p>
							<p className="text-foreground text-sm font-mono font-semibold">
								{sessionsByTrack[trackName].length} session
								{sessionsByTrack[trackName].length !== 1 ? "s" : ""}
							</p>
						</div>
					))}
				</div>
			)}
		</div>
	);
}

export const Route = createFileRoute("/")({
	component: RouteComponent,
});
