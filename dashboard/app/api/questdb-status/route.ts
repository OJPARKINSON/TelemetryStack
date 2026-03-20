import { NextResponse } from "next/server";
import {
	getQuestDBHealth,
	getQuestDBStats,
	getSessions,
} from "../../../lib/questDb";

export async function GET() {
	const startTime = Date.now();

	try {
		console.log("🔍 Running comprehensive QuestDB status check...");

		// Run health check
		const healthResult = await getQuestDBHealth();

		// Get database stats
		const stats = await getQuestDBStats();

		// Try to get sessions (but don't fail if it doesn't work)
		let sessionsResult = null;
		try {
			const sessions = await getSessions();
			sessionsResult = {
				count: sessions.length,
				latest: sessions.slice(0, 3).map((s) => ({
					id: s.session_id,
					updated: s.last_updated.toISOString(),
				})),
			};
		} catch (error) {
			sessionsResult = {
				error: error instanceof Error ? error.message : String(error),
			};
		}

		const totalTime = Date.now() - startTime;

		const response = {
			timestamp: new Date().toISOString(),
			totalResponseTime: totalTime,
			health: healthResult,
			database: stats,
			sessions: sessionsResult,
			environment: {
				// nodeEnv: import.meta.env.MODE,
				// questdbHost: process.env.QUESTDB_HOST || "default",
				// questdbPort: process.env.QUESTDB_PORT || "default",
			},
		};

		console.log("✅ QuestDB status check completed in", totalTime, "ms");

		if (healthResult.healthy) {
			return NextResponse.json(response);
		}
		return NextResponse.json(response, { status: 503 });
	} catch (error) {
		console.error("❌ QuestDB status check failed:", error);

		const errorResponse = {
			timestamp: new Date().toISOString(),
			totalResponseTime: Date.now() - startTime,
			error: error instanceof Error ? error.message : String(error),
			healthy: false,
		};

		return NextResponse.json(errorResponse, { status: 500 });
	}
}
