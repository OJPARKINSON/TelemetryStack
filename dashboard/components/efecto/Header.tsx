import { Link } from "@tanstack/react-router";
import { Bar } from "./Skeleton";

interface HeaderProps {
	/** Undefined while loading. */
	trackName?: string;
	sessionNum: string;
	currentLapId: number;
	/** Undefined while the lap list loads. */
	totalLaps?: number;
	/** Undefined while loading. */
	lapTime?: string;
	onPrevLap: () => void;
	onNextLap: () => void;
	onLapChange: (lapId: number) => void;
}

export const Header = ({
	trackName,
	sessionNum,
	currentLapId,
	totalLaps,
	lapTime,
	onPrevLap,
	onNextLap,
	onLapChange,
}: HeaderProps) => {
	// Commit on Enter/blur, not per keystroke, so typing "20" doesn't load lap 2 first
	const commitLap = (input: HTMLInputElement) => {
		const lap = Math.round(Number(input.value));
		if (totalLaps && lap >= 1 && lap <= totalLaps && lap !== currentLapId) {
			onLapChange(lap);
		} else {
			input.value = String(currentLapId);
		}
	};

	return (
		<div className="flex items-center justify-between px-8 py-4 border-b border-border w-full bg-background">
			<div className="flex items-center gap-6">
				<Link to="/">
					<div className="flex items-center gap-2.5">
						<div className="w-7 h-7 rounded-md bg-primary flex items-center justify-center">
							<p className="text-primary-foreground text-xs font-bold">iR</p>
						</div>
						<p className="text-foreground text-sm font-medium tracking-tight">
							Telemetry
						</p>
						<p className="text-muted-foreground text-sm">/</p>
						<p className="text-muted-foreground text-sm">
							{trackName ?? <Bar className="w-20" />}
						</p>
					</div>
				</Link>
			</div>
			<div className="flex items-center gap-4">
				<div className="flex items-center gap-0 rounded-lg border border-border overflow-hidden bg-card">
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
							<input
								key={currentLapId}
								type="number"
								inputMode="numeric"
								aria-label="Lap number"
								min={1}
								max={totalLaps}
								defaultValue={currentLapId}
								disabled={!totalLaps}
								onFocus={(e) => e.currentTarget.select()}
								onBlur={(e) => commitLap(e.currentTarget)}
								onKeyDown={(e) => {
									if (e.key === "Enter") commitLap(e.currentTarget);
									if (e.key === "Escape") {
										e.currentTarget.value = String(currentLapId);
										e.currentTarget.blur();
									}
								}}
								className="w-[4ch] bg-secondary rounded px-1 py-0.5 border border-border text-foreground text-xs font-mono font-semibold text-center outline-none focus:border-primary [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
							/>
							<p className="text-muted-foreground text-xs font-mono min-w-[4ch]">
								/ {totalLaps ?? <Bar className="w-[2ch]" />}
							</p>
						</div>
						<div className="h-3 w-px bg-border" />
						<p className="text-emerald-500 text-xs font-mono font-semibold min-w-[7ch]">
							{lapTime ?? <Bar className="w-[7ch]" />}
						</p>
					</div>
					<button
						type="button"
						onClick={onNextLap}
						disabled={currentLapId >= (totalLaps ?? 0)}
						className="px-2 py-1.5 flex items-center justify-center border-l border-border hover:bg-secondary transition-colors disabled:opacity-30"
					>
						<p className="text-muted-foreground text-xs">→</p>
					</button>
				</div>
				<div className="flex items-center gap-2">
					<div className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
					<p className="text-muted-foreground text-xs whitespace-nowrap">
						Session #{sessionNum}
					</p>
				</div>
			</div>
		</div>
	);
};
