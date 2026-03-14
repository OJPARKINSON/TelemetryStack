import { Link } from "@tanstack/react-router";

export function DatabaseUnavailableError() {
	return (
		<div>
			<div className="mx-auto max-w-md p-8 text-center">
				<div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-red-500/20">
					<div className="h-8 w-8 text-red-400">
						<svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<title>Error</title>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"
							/>
						</svg>
					</div>
				</div>
				<h1 className="mb-4 font-bold text-2xl text-white">
					Database Unavailable
				</h1>
				<p className="mb-6 text-zinc-400">
					The telemetry database is not running or accessible. Please start the
					Docker Compose stack to access telemetry data.
				</p>
				<div className="rounded-lg bg-zinc-800/50 p-4 text-left">
					<h3 className="mb-2 font-semibold text-sm text-zinc-300">
						To start the system:
					</h3>
					<code className="block rounded bg-zinc-900 px-2 py-1 text-xs text-zinc-400">
						docker compose up -d
					</code>
				</div>
				<div className="mt-6">
					<Link
						to=".."
						className="inline-flex items-center rounded-lg bg-blue-600 px-4 py-2 text-white transition-colors hover:bg-blue-700"
					>
						← Back to Dashboard
					</Link>
				</div>
			</div>
		</div>
	);
}
