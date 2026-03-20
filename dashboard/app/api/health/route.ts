import { NextResponse } from "next/server";
import { getQuestDBHealth } from "../../../lib/questDb";

export async function GET() {
	try {
		const healthResult = await getQuestDBHealth();

		if (healthResult.healthy) {
			return NextResponse.json({
				status: "ok",
				timestamp: new Date().toISOString(),
				message: healthResult.message,
				database: "QuestDB",
				details: healthResult.details,
			});
		}
		return NextResponse.json(
			{
				status: "error",
				message: healthResult.message,
				timestamp: new Date().toISOString(),
				database: "QuestDB",
				details: healthResult.details,
			},
			{ status: 503 },
		);
	} catch (error) {
		console.error("Health check failed:", error);
		return NextResponse.json(
			{
				status: "error",
				message: "Health check failed",
				timestamp: new Date().toISOString(),
				error: error instanceof Error ? error.message : String(error),
				database: "QuestDB",
			},
			{ status: 503 },
		);
	}
}
