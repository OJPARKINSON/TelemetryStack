interface StatBoxProps {
	title: string;
	stat: string;
	unit?: string;
	delta?: string;
}

export const StatBox = ({ title, stat, unit, delta }: StatBoxProps) => {
	return (
		<div className="flex-1 flex flex-col justify-center px-8 py-2 border-r border-border last:border-r-0">
			<p className="text-muted-foreground text-xs font-medium mb-1">{title}</p>
			<div className="flex items-baseline gap-2">
				<p className="text-foreground text-xl font-mono font-semibold tracking-tight">
					{stat}
				</p>
				{unit && (
					<p className="text-muted-foreground text-xs font-mono">{unit}</p>
				)}
				{delta && <p className="text-emerald-500 text-xs font-mono">{delta}</p>}
			</div>
		</div>
	);
};
