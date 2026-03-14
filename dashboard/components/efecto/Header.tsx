interface HeaderProps {
	trackName: string;
	sessionNum: string;
	currentLapId: number;
	totalLaps: number;
	lapTime: string;
	onPrevLap: () => void;
	onNextLap: () => void;
}

export const Header = ({
	trackName,
	sessionNum,
	currentLapId,
	totalLaps,
	lapTime,
	onPrevLap,
	onNextLap,
}: HeaderProps) => {
	return (
		<div className="flex items-center justify-between px-8 py-4 border-b border-border w-full bg-background">
			<div className="flex items-center gap-6">
				<div className="flex items-center gap-2.5">
					<div className="w-7 h-7 rounded-md bg-primary flex items-center justify-center">
						<p className="text-primary-foreground text-xs font-bold">iR</p>
					</div>
					<p className="text-foreground text-sm font-medium tracking-tight">
						Telemetry
					</p>
					<p className="text-muted-foreground text-sm">/</p>
					<p className="text-muted-foreground text-sm">{trackName}</p>
				</div>
				<div className="flex items-center gap-1 ml-4">
					<div className="px-3 py-1.5 rounded-md bg-secondary">
						<p className="text-foreground text-xs font-medium">Lap Data</p>
					</div>
					<div className="px-3 py-1.5 rounded-md">
						<p className="text-muted-foreground text-xs font-medium">
							Track Map
						</p>
					</div>
					<div className="px-3 py-1.5 rounded-md">
						<p className="text-muted-foreground text-xs font-medium">Sectors</p>
					</div>
					<div className="px-3 py-1.5 rounded-md">
						<p className="text-muted-foreground text-xs font-medium">Compare</p>
					</div>
				</div>
			</div>
			<div className="flex items-center gap-4">
				<div className="flex items-center gap-0 rounded-lg border border-border overflow-hidden bg-card w-full">
					<button
						type="button"
						onClick={onPrevLap}
						disabled={currentLapId <= 1}
						className="px-2 py-1.5 flex items-center justify-center border-r border-border hover:bg-secondary transition-colors disabled:opacity-30"
					>
						<p className="text-muted-foreground text-xs">←</p>
					</button>
					<div className="flex items-center gap-2.5 px-2.5 py-1">
						<div className="flex items-baseline gap-1">
							<p className="text-muted-foreground text-xs">Lap</p>
							<div className="bg-secondary rounded px-1 py-0.5 border border-border">
								<p className="text-foreground text-xs font-mono font-semibold">
									{currentLapId}
								</p>
							</div>
							<p className="text-muted-foreground text-xs font-mono">
								/ {totalLaps}
							</p>
						</div>
						<div className="h-3 w-px bg-border" />
						<p className="text-emerald-500 text-xs font-mono font-semibold">{lapTime}</p>
					</div>
					<button
						type="button"
						onClick={onNextLap}
						disabled={currentLapId >= totalLaps}
						className="px-2 py-1.5 flex items-center justify-center border-l border-border hover:bg-secondary transition-colors disabled:opacity-30"
					>
						<p className="text-muted-foreground text-xs">→</p>
					</button>
				</div>
				<div className="flex items-center gap-2">
					<div className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
					<p className="text-muted-foreground text-xs">Session #{sessionNum}</p>
				</div>
			</div>
		</div>
	);
};
