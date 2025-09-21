export default function Loading() {
	return (
		<div className="min-h-screen bg-zinc-950 flex">
			{/* Sidebar Skeleton */}
			<div className="w-64 bg-zinc-900/50 border-r border-zinc-800/50 flex flex-col">
				<div className="px-6 py-6">
					<div className="animate-pulse">
						<div className="flex items-center space-x-3">
							<div className="w-8 h-8 bg-zinc-700 rounded-lg"></div>
							<div>
								<div className="h-4 w-16 bg-zinc-700 rounded mb-1"></div>
								<div className="h-3 w-12 bg-zinc-700 rounded"></div>
							</div>
						</div>
					</div>
				</div>

				{/* Navigation skeleton */}
				<nav className="flex-1 px-4 space-y-1">
					<div className="animate-pulse space-y-2">
						<div className="h-8 bg-zinc-800/50 rounded-md"></div>
						<div className="h-8 bg-zinc-800/50 rounded-md"></div>
						<div className="h-8 bg-zinc-800/50 rounded-md"></div>
					</div>
				</nav>
			</div>

			<div className="flex-1 flex flex-col">
				{/* Header skeleton */}
				<div className="bg-zinc-950/50 border-b border-zinc-800/50 px-6 py-4">
					<div className="animate-pulse h-4 w-48 bg-zinc-700 rounded"></div>
				</div>

				{/* Main content skeleton */}
				<div className="flex-1 p-6">
					<div className="animate-pulse space-y-6">
						{/* Stats cards */}
						<div className="bg-zinc-900/50 border border-zinc-800/50 rounded-lg p-4">
							<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
								{[...Array(4)].map((_, i) => (
									// biome-ignore lint/suspicious/noArrayIndexKey: na
									<div key={i} className="bg-zinc-800/50 h-20 rounded-lg"></div>
								))}
							</div>
						</div>

						{/* Main grid layout */}
						<div className="grid grid-cols-1 lg:grid-cols-5 gap-6">
							{/* Track map area - this will be LCP */}
							<div className="col-span-1 lg:col-span-3 bg-zinc-900/50 border border-zinc-800/50 rounded-lg p-6">
								<div className="h-[500px] bg-zinc-800/30 rounded-lg flex items-center justify-center">
									<div className="text-center">
										<div className="w-16 h-16 mx-auto bg-zinc-700/50 rounded-lg flex items-center justify-center mb-4">
											<div className="w-8 h-8 border-2 border-zinc-600 rounded border-dashed animate-spin"></div>
										</div>
										<p className="text-zinc-400 text-sm">
											Loading telemetry data...
										</p>
									</div>
								</div>
							</div>

							{/* Charts area */}
							<div className="col-span-1 lg:col-span-2 bg-zinc-900/50 border border-zinc-800/50 rounded-lg p-4">
								<div className="h-[600px] bg-zinc-800/30 rounded-lg"></div>
							</div>
						</div>

						{/* Info box skeleton */}
						<div className="bg-zinc-900/50 border border-zinc-800/50 rounded-lg p-6">
							<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
								{[...Array(4)].map((_, i) => (
									// biome-ignore lint/suspicious/noArrayIndexKey: na
									<div key={i} className="bg-zinc-800/50 h-20 rounded-lg"></div>
								))}
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	);
}
