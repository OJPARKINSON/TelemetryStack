import { Bar } from "./Skeleton";

interface StatBoxProps {
	title: string;
	stat: string;
	unit?: string;
	delta?: string;
	/** Render a placeholder in place of the value, keeping the same box size. */
	loading?: boolean;
}

export const StatBox = ({
	title,
	stat,
	unit,
	delta,
	loading,
}: StatBoxProps) => {
	return (
		<div className="flex-1 flex flex-col justify-center px-8 py-2 border-r border-border last:border-r-0">
			<p className="text-muted-foreground text-xs font-medium mb-1">{title}</p>
			<div className="flex items-baseline gap-2">
				<p className="text-foreground text-xl font-mono font-semibold tracking-tight">
					{loading ? <Bar className="w-[6ch]" /> : stat}
				</p>
				{!loading && unit && (
					<p className="text-muted-foreground text-xs font-mono">{unit}</p>
				)}
				{!loading && delta && (
					<p className="text-emerald-500 text-xs font-mono">{delta}</p>
				)}
			</div>
		</div>
	);
};
