export default function Loading() {
	return (
		<div className="flex min-h-screen bg-zinc-950">
			{/* Sidebar Skeleton */}
			<div className="flex w-64 flex-col border-zinc-800/50 border-r bg-zinc-900/50">
				<div className="px-6 py-6">
					<div className="animate-pulse">
						<div className="flex items-center space-x-3">
							<div className="h-8 w-8 rounded-lg bg-zinc-700" />
							<div>
								<div className="mb-1 h-4 w-16 rounded bg-zinc-700" />
								<div className="h-3 w-12 rounded bg-zinc-700" />
							</div>
						</div>
					</div>
				</div>

				{/* Navigation skeleton */}
				<nav className="flex-1 space-y-1 px-4">
					<div className="animate-pulse space-y-2">
						<div className="h-8 rounded-md bg-zinc-800/50" />
						<div className="h-8 rounded-md bg-zinc-800/50" />
						<div className="h-8 rounded-md bg-zinc-800/50" />
					</div>
				</nav>
			</div>

			<div className="flex flex-1 flex-col">
				{/* Header skeleton */}
				<div className="border-zinc-800/50 border-b bg-zinc-950/50 px-6 py-4">
					<div className="h-4 w-48 animate-pulse rounded bg-zinc-700" />
				</div>

				{/* Main content skeleton */}
				<div className="flex-1 p-6">
					<div className="animate-pulse space-y-6">
						{/* Stats cards */}
						<div className="rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-4">
							<div className="grid grid-cols-2 gap-4 md:grid-cols-4">
								{[...Array(4)].map((_, i) => (
									// biome-ignore lint/suspicious/noArrayIndexKey: na
									<div key={i} className="h-20 rounded-lg bg-zinc-800/50" />
								))}
							</div>
						</div>

						{/* Main grid layout */}
						<div className="grid grid-cols-1 gap-6 lg:grid-cols-5">
							{/* Track map area - this will be LCP */}
							<div className="col-span-1 rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-6 lg:col-span-3">
								<div className="flex h-[500px] items-center justify-center rounded-lg bg-zinc-800/30">
									<div className="text-center">
										<div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-lg bg-zinc-700/50">
											<div className="h-8 w-8 animate-spin rounded border-2 border-zinc-600 border-dashed" />
										</div>
										<p className="text-sm text-zinc-400">
											Loading telemetry data...
										</p>
									</div>
								</div>
							</div>

							{/* Charts area */}
							<div className="col-span-1 rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-4 lg:col-span-2">
								<div className="h-[600px] rounded-lg bg-zinc-800/30" />
							</div>
						</div>

						{/* Info box skeleton */}
						<div className="rounded-lg border border-zinc-800/50 bg-zinc-900/50 p-6">
							<div className="grid grid-cols-2 gap-4 md:grid-cols-4">
								{[...Array(4)].map((_, i) => (
									// biome-ignore lint/suspicious/noArrayIndexKey: na
									<div key={i} className="h-20 rounded-lg bg-zinc-800/50" />
								))}
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	);
}
