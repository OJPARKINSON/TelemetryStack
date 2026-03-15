/** biome-ignore-all lint/suspicious/noArrayIndexKey: <explanation> */
export function NewTelemetrySkeleton() {
	return (
		<div className="flex flex-col bg-background border border-border w-full mx-auto min-h-screen">
			{/* Header skeleton */}
			<div className="flex items-center justify-between px-8 py-4 border-b border-border">
				<div className="flex items-center gap-6">
					<div className="flex items-center gap-2.5">
						<div className="w-7 h-7 rounded-md bg-secondary animate-pulse" />
						<div className="h-4 w-16 rounded bg-secondary animate-pulse" />
						<div className="h-4 w-24 rounded bg-secondary animate-pulse" />
					</div>
					<div className="flex items-center gap-1 ml-4">
						<div className="h-7 w-16 rounded-md bg-secondary animate-pulse" />
						<div className="h-7 w-16 rounded-md bg-secondary/50 animate-pulse" />
						<div className="h-7 w-14 rounded-md bg-secondary/50 animate-pulse" />
						<div className="h-7 w-16 rounded-md bg-secondary/50 animate-pulse" />
					</div>
				</div>
				<div className="flex items-center gap-4">
					<div className="h-8 w-48 rounded-lg border border-border bg-secondary animate-pulse" />
					<div className="h-4 w-24 rounded bg-secondary animate-pulse" />
				</div>
			</div>

			{/* Stat boxes skeleton */}
			<div className="flex items-stretch border-b border-border">
				{/* biome-ignore lint/suspicious/noArrayIndexKey: skeleton placeholders */}
				{Array.from({ length: 5 }).map((_, i) => (
					<div
						key={`stat-${i}`}
						className="flex-1 px-8 py-5 border-r border-border last:border-r-0"
					>
						<div className="h-3 w-16 rounded bg-secondary animate-pulse mb-2" />
						<div className="h-7 w-24 rounded bg-secondary animate-pulse" />
					</div>
				))}
			</div>

			{/* Main content skeleton */}
			<div className="flex flex-row flex-1">
				{/* Map area */}
				<div className="flex flex-col border-r border-border flex-1 min-w-0">
					<div className="flex items-center justify-between px-6 py-3 border-b border-border">
						<div className="h-3 w-20 rounded bg-secondary animate-pulse" />
						<div className="h-3 w-32 rounded bg-secondary animate-pulse" />
					</div>
					<div className="flex-1 bg-card min-h-[500px] flex items-center justify-center">
						<div className="w-64 h-64 rounded-full border-2 border-secondary animate-pulse" />
					</div>
					<div className="flex items-center gap-2 px-6 py-3 border-t border-border">
						<div className="h-7 w-16 rounded-md bg-secondary animate-pulse" />
						{/* biome-ignore lint/suspicious/noArrayIndexKey: skeleton placeholders */}
						{Array.from({ length: 3 }).map((_, i) => (
							<div
								key={`sector-${i}`}
								className="h-7 w-24 rounded-md bg-secondary/50 animate-pulse"
							/>
						))}
					</div>
				</div>

				{/* Charts area */}
				<div className="flex flex-col min-w-[400px]">
					<div className="flex items-center justify-between px-5 py-3 border-b border-border">
						<div className="h-3 w-16 rounded bg-secondary animate-pulse" />
					</div>
					<div className="flex-1 overflow-y-auto px-2 py-2 space-y-2">
						{/* biome-ignore lint/suspicious/noArrayIndexKey: skeleton placeholders */}
						{Array.from({ length: 6 }).map((_, i) => (
							<div
								key={`chart-${i}`}
								className="rounded-lg bg-secondary/30 px-3 py-2"
							>
								<div className="flex items-center justify-between mb-1">
									<div className="h-3 w-12 rounded bg-secondary animate-pulse" />
									<div className="h-3 w-8 rounded bg-secondary animate-pulse" />
								</div>
								<div
									className="bg-secondary/20 rounded animate-pulse"
									style={{ height: i === 0 ? 120 : i === 3 ? 80 : 100 }}
								/>
							</div>
						))}
					</div>
					<div className="flex items-center justify-between px-5 py-2 border-t border-border">
						<div className="h-3 w-6 rounded bg-secondary animate-pulse" />
						<div className="h-3 w-20 rounded bg-secondary animate-pulse" />
						<div className="h-3 w-8 rounded bg-secondary animate-pulse" />
					</div>
				</div>
			</div>

			{/* Footer skeleton */}
			<div className="flex items-stretch border-t border-border">
				{/* biome-ignore lint/suspicious/noArrayIndexKey: skeleton placeholders */}
				{Array.from({ length: 5 }).map((_, i) => (
					<div
						key={`footer-${i}`}
						className="flex-1 px-6 py-3 border-r border-border last:border-r-0"
					>
						<div className="h-3 w-16 rounded bg-secondary animate-pulse mb-1" />
						<div className="h-4 w-20 rounded bg-secondary animate-pulse" />
					</div>
				))}
			</div>
		</div>
	);
}
