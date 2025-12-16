"use client";

import Link from "next/link";

export interface Session {
	session_id: string;
	last_updated: Date;
	track_name: string;
}

interface SessionSelectorProps {
	sessions: Session[];
}

export default function SessionSelector({ sessions }: SessionSelectorProps) {
	const formatDate = (date: Date) => {
		return new Intl.DateTimeFormat("en-US", {
			year: "numeric",
			month: "short",
			day: "numeric",
			hour: "2-digit",
			minute: "2-digit",
		}).format(new Date(date));
	};

	const formatRelativeTime = (date: Date) => {
		const now = new Date();
		const diffMs = now.getTime() - new Date(date).getTime();
		const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
		const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
		const diffMinutes = Math.floor(diffMs / (1000 * 60));

		if (diffDays > 0) return `${diffDays}d ago`;
		if (diffHours > 0) return `${diffHours}h ago`;
		if (diffMinutes > 0) return `${diffMinutes}m ago`;
		return "Just now";
	};

	// Group sessions by track for better organization
	const sessionsByTrack = sessions.reduce(
		(acc, session) => {
			const track = session.track_name || "Unknown Track";
			if (!acc[track]) acc[track] = [];
			acc[track].push(session);
			return acc;
		},
		{} as Record<string, Session[]>,
	);

	return (
		<div className="space-y-6">
			{/* Sessions Grid */}
			<div className="grid grid-cols-1 gap-4">
				{sessions.map((session) => (
					<Link
						key={session.session_id}
						href={`/${session.session_id}?lap=1`}
						className="group rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-6 text-left transition-all duration-200 hover:border-zinc-700/50"
					>
						<div className="flex items-center justify-between">
							<div className="flex-1">
								<div className="mb-4 flex items-start justify-between">
									<div>
										<h3 className="font-semibold text-lg text-white transition-colors group-hover:text-blue-400">
											{session.session_id}
										</h3>
										<p className="mt-1 text-sm text-zinc-400">
											{session.track_name || "Unknown Track"}
										</p>
									</div>
									<div className="text-right">
										<div className="mb-1 flex items-center space-x-2">
											<div className="h-2 w-2 rounded-full bg-green-400" />
											<span className="text-xs text-zinc-400">Ready</span>
										</div>
										<p className="text-xs text-zinc-500">
											{formatRelativeTime(session.last_updated)}
										</p>
									</div>
								</div>

								<div className="flex items-center justify-between text-sm">
									<div className="flex items-center space-x-4">
										<div>
											<span className="text-zinc-400">Date: </span>
											<span className="text-white">
												{formatDate(session.last_updated)}
											</span>
										</div>
									</div>
									<div className="flex items-center text-zinc-400 transition-colors group-hover:text-white">
										<span className="text-xs">Analyze session</span>
										{/** biome-ignore lint/a11y/noSvgWithoutTitle: na */}
										<svg
											className="ml-2 h-4 w-4 opacity-0 transition-opacity group-hover:opacity-100"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
										>
											<path
												strokeLinecap="round"
												strokeLinejoin="round"
												strokeWidth={2}
												d="M9 5l7 7-7 7"
											/>
										</svg>
									</div>
								</div>
							</div>
						</div>
					</Link>
				))}
			</div>

			{/* Quick Filter */}
			{Object.keys(sessionsByTrack).length > 1 && (
				<div className="border-zinc-800/50 border-t pt-6">
					<h3 className="mb-4 font-medium text-sm text-zinc-300">
						Filter by Track
					</h3>
					<div className="flex flex-wrap gap-2">
						{Object.entries(sessionsByTrack).map(
							([trackName, trackSessions]) => (
								<div
									key={trackName}
									className="rounded-lg border border-zinc-800/50 bg-zinc-900/30 px-3 py-2"
								>
									<div className="flex items-center space-x-2">
										<span className="text-sm text-white">{trackName}</span>
										<span className="rounded bg-zinc-800/50 px-2 py-0.5 text-xs text-zinc-400">
											{trackSessions.length}
										</span>
									</div>
								</div>
							),
						)}
					</div>
				</div>
			)}
		</div>
	);
}
