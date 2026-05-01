import { Link } from "@tanstack/react-router";

export interface Session {
	last_updated: string;
	max_lap_id: string;
	session_id: string;
	session_name: string;
	track_name: string;
}

interface SessionSelectorProps {
	sessions: Session[];
}

function getTrackAbbrev(trackName: string): string {
	if (!trackName) return "??";
	const words = trackName.split(/[\s-]+/);
	if (words.length === 1) return words[0].slice(0, 2).toUpperCase();
	return (words[0][0] + words[1][0]).toUpperCase();
}

function formatDate(dateStr: string): string {
	return new Intl.DateTimeFormat("en-US", {
		month: "short",
		day: "numeric",
		year: "numeric",
	}).format(new Date(dateStr));
}

function formatRelativeTime(dateStr: string): string {
	const now = new Date();
	const diffMs = now.getTime() - new Date(dateStr).getTime();
	const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
	const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
	const diffMinutes = Math.floor(diffMs / (1000 * 60));

	if (diffDays > 0) return `${diffDays} days ago`;
	if (diffHours > 0) return `${diffHours}h ago`;
	if (diffMinutes > 0) return `${diffMinutes}m ago`;
	return "Just now";
}

export default function SessionSelector({ sessions }: SessionSelectorProps) {
	return (
		<div className="flex flex-col">
			{sessions.map((session) => (
				<Link
					key={session.session_id}
					to="/$sessionId"
					params={{ sessionId: session.session_id }}
					search={{ lapId: 1 }}
					className="group flex items-center px-8 py-3.5 border-b border-border hover:bg-secondary/50 transition-colors"
				>
					<div className="flex items-center gap-3 flex-1">
						<div className="w-8 h-8 rounded-md bg-secondary flex items-center justify-center border border-border">
							<span className="text-foreground text-xs font-semibold">
								{getTrackAbbrev(session.track_name)}
							</span>
						</div>
						<div className="flex flex-col">
							<span className="text-foreground text-sm font-medium tracking-tight">
								{session.track_name || "Unknown Track"}
							</span>
							<span className="text-muted-foreground text-xs">
								Session #{session.session_id}
							</span>
						</div>
					</div>
					<div className="flex items-center gap-8">
						<div className="flex items-center gap-1.5">
							<span className="text-foreground text-sm font-mono font-semibold">
								{session.max_lap_id}
							</span>
							<span className="text-muted-foreground text-xs">laps</span>
						</div>
						<div className="flex items-center gap-2">
							<span className="text-muted-foreground text-xs">
								{formatDate(session.last_updated)}
							</span>
							<div className="h-3 w-px bg-border" />
							<span className="text-muted-foreground text-xs">
								{formatRelativeTime(session.last_updated)}
							</span>
						</div>
						<div className="px-3 py-1.5 rounded-md bg-secondary border border-border group-hover:bg-primary group-hover:text-primary-foreground group-hover:border-primary transition-colors">
							<span className="text-foreground text-xs font-medium group-hover:text-primary-foreground">
								Analyze
							</span>
						</div>
					</div>
				</Link>
			))}
		</div>
	);
}
