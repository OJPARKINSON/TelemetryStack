import { Link } from "@tanstack/react-router";

interface TelemetrySidebarProps {
	availableLaps?: number[];
	currentLapId: number;
	sessionId: string;
	onLapChange: (lapId: string) => void;
}

export function TelemetrySidebar({
	availableLaps,
	currentLapId,
	sessionId,
	onLapChange,
}: TelemetrySidebarProps) {
	return (
		<div className="flex w-48 flex-col border-zinc-800/50 border-r bg-zinc-900/50">
			<div className="px-6 py-6">
				<Link to="/" className="cursor-pointer">
					<div className="flex h-8 w-8 items-center justify-center rounded-lg bg-white">
						<div className="h-4 w-4 rounded bg-zinc-900" />
					</div>
					<div className="flex items-center space-x-3">
						<div>
							<h1 className="font-semibold text-sm text-white">iRacing</h1>
							<p className="text-xs text-zinc-400">Telemetry</p>
						</div>
					</div>
				</Link>
			</div>

			<nav className="flex-1 space-y-1 px-4">
				<div className="flex cursor-pointer items-center justify-between rounded-md px-3 py-2 font-medium text-sm text-white">
					Lap:
					<select
						className="h-fit rounded border border-zinc-600 bg-zinc-800/90 px-3 py-1 font-medium text-sm text-white hover:bg-zinc-700/90 focus:outline-none focus:ring-2 focus:ring-blue-500"
						onChange={(e) => onLapChange(e.currentTarget.value)}
						value={(currentLapId || "")?.toString()}
					>
						{availableLaps?.map((lap) => (
							<option key={lap.toString()} value={lap.toString()}>
								{lap.toString()}
							</option>
						))}
					</select>
				</div>
				<div className="px-2 py-2 font-medium text-xs text-zinc-500 uppercase tracking-wider">
					Analysis
				</div>
				<div className="rounded-md bg-zinc-800/50 px-3 py-2 font-medium text-sm text-white">
					Session {sessionId}
				</div>
				<div className="cursor-pointer rounded-md px-3 py-2 font-medium text-sm text-zinc-400 hover:bg-zinc-800/50 hover:text-white">
					Track Map
				</div>
				<div className="cursor-pointer rounded-md px-3 py-2 font-medium text-sm text-zinc-400 hover:bg-zinc-800/50 hover:text-white">
					Telemetry
				</div>
				<div className="cursor-pointer rounded-md px-3 py-2 font-medium text-sm text-zinc-400 hover:bg-zinc-800/50 hover:text-white">
					Performance
				</div>
			</nav>
		</div>
	);
}
