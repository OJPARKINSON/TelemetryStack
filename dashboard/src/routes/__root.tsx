import { createRootRoute, Outlet } from "@tanstack/react-router";

export const Route = createRootRoute({
	component: DashboardPage,
});

export default function DashboardPage() {
	return (
		<div className="flex min-h-screen min-w-screen bg-zinc-950">
			<Outlet />
		</div>
	);
}
