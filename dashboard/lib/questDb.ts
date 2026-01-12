// import { Client } from "pg";
import type { Session } from "@/components/SessionSelector";
import { processIRacingDataWithGPS, type TelemetryRes } from "./Fetch";

interface QuestDBConfig {
	host: string;
	port: number;
	user?: string;
	password?: string;
	database?: string;
}

class QuestDBClient {
	private config: QuestDBConfig;

	constructor() {
		this.config = {
			host:
				import.meta.env.MODE ||
				(import.meta.env.MODE === "production" ? "questdb" : "127.0.0.1"),
			port: Number.parseInt(import.meta.env.VITE_QUESTDB_PORT || "8812", 10),
			user: import.meta.env.VITE_QUESTDB_USER || "admin",
			password: import.meta.env.VITE_QUESTDB_PASSWORD || "quest",
			database: import.meta.env.VITE_QUESTDB_DATABASE || "qdb",
		};
	}

	private createClient(): Client {
		const client = new Client({
			host: this.config.host,
			port: this.config.port,
			user: this.config.user,
			password: this.config.password,
			database: this.config.database,
			ssl: false,
			connectionTimeoutMillis: 15000,
			query_timeout: 30000,
			statement_timeout: 30000,
			keepAlive: true,
		});

		return client;
	}

	private async executeWithRetry<T>(
		operation: (client: Client) => Promise<T>,
		maxRetries = 3,
		baseDelay = 1000,
	): Promise<T> {
		let lastError: Error | null = null;

		for (let attempt = 1; attempt <= maxRetries; attempt++) {
			const client = this.createClient();

			try {
				console.log(
					`🔌 Attempt ${attempt}: Connecting to QuestDB at ${this.config.host}:${this.config.port}`,
				);

				await client.connect();
				console.log(
					`✅ Connected to QuestDB successfully (attempt ${attempt})`,
				);

				const result = await operation(client);
				console.log("✅ Operation completed successfully");

				return result;
			} catch (error) {
				lastError = error instanceof Error ? error : new Error(String(error));
				console.error(`❌ Attempt ${attempt} failed:`, lastError.message);

				if (attempt === maxRetries) {
					break;
				}

				// Exponential backoff
				const delay = baseDelay * 2 ** (attempt - 1);
				console.log(`⏳ Waiting ${delay}ms before retry...`);
				await new Promise((resolve) => setTimeout(resolve, delay));
			} finally {
				try {
					await client.end();
				} catch (e) {
					console.warn("Warning: Error closing connection:", e);
				}
			}
		}

		throw new Error(
			`QuestDB operation failed after ${maxRetries} attempts: ${lastError?.message}`,
		);
	}

	async getTelemetryData(
		sessionId: string,
		lapId: number,
	): Promise<TelemetryRes | null> {
		try {
			console.log(
				`📊 Fetching telemetry data for session: ${sessionId}, lap: ${lapId}`,
			);

			return await this.executeWithRetry(async (client) => {
				const query = `
                    SELECT * FROM TelemetryTicks 
                    WHERE session_id = $1 AND lap_id = $2 AND session_type = 'Race'
                    ORDER BY session_time ASC
                `;

				const result = await client.query(query, [sessionId, lapId]);

				if (result.rowCount === 0) {
					console.log(
						`No telemetry data found for session ${sessionId}, lap ${lapId}`,
					);
					return null;
				}

				console.log(`Found ${result.rowCount} telemetry data points`);
				return processIRacingDataWithGPS(result);
			});
		} catch (error) {
			console.error("Failed to fetch telemetry data:", error);
			throw new Error(
				`Failed to fetch telemetry data: ${error instanceof Error ? error.message : String(error)}`,
			);
		}
	}

	async getLaps(sessionId: string): Promise<Array<{ lap_id: number }>> {
		try {
			console.log(`📋 Fetching laps for session: ${sessionId}`);

			return await this.executeWithRetry(async (client) => {
				const query = `
                    SELECT DISTINCT lap_id 
                    FROM TelemetryTicks 
                    WHERE session_id = $1 
                    ORDER BY lap_id ASC
                `;

				const result = await client.query(query, [sessionId]);

				const laps = result.rows
					.map((row) => ({
						lap_id: Number.parseInt(row.lap_id, 10),
					}))
					.sort(({ lap_id }, b) => lap_id - b.lap_id);

				console.log(`Found ${laps.length} laps for session ${sessionId}`);
				return laps;
			});
		} catch (error) {
			console.error("Failed to fetch laps:", error);
			throw new Error(
				`Failed to fetch laps: ${error instanceof Error ? error.message : String(error)}`,
			);
		}
	}

	async getSessions(): Promise<Session[]> {
		try {
			return await this.executeWithRetry(async (client) => {
				const query = `
				SELECT DISTINCT session_id, 
				track_name, session_name, MAX(lap_id), MAX(timestamp) as last_updated
				FROM TelemetryTicks 
       			WHERE session_name = 'RACE' and lap_id > 0
				ORDER BY last_updated DESC
                `;

				const result = await client.query(query);

				const sessions = result.rows.map((row) => ({
					session_id: row.session_id,
					last_updated: new Date(row.last_updated),
					track_name: row.track_name,
				}));

				console.log(`Found ${sessions.length} sessions`);
				return sessions;
			});
		} catch (error) {
			console.error("Failed to fetch sessions:", error);
			throw new Error(
				`Failed to fetch sessions: ${error instanceof Error ? error.message : String(error)}`,
			);
		}
	}

	async healthCheck(): Promise<{
		healthy: boolean;
		message: string;
		details?: any;
	}> {
		try {
			console.log("🏥 Performing QuestDB health check");

			const result = await this.executeWithRetry(
				async (client) => {
					const startTime = Date.now();
					await client.query("SELECT 1 as health_check");
					const responseTime = Date.now() - startTime;

					return { responseTime };
				},
				2,
				500,
			); // Fewer retries for health check

			return {
				healthy: true,
				message: "QuestDB is accessible",
				details: {
					host: this.config.host,
					port: this.config.port,
					responseTime: result.responseTime,
				},
			};
		} catch (error) {
			return {
				healthy: false,
				message: `QuestDB is not accessible: ${error instanceof Error ? error.message : String(error)}`,
				details: {
					host: this.config.host,
					port: this.config.port,
					error: error instanceof Error ? error.message : String(error),
				},
			};
		}
	}

	/**
	 * Get database statistics
	 */
	async getStats(): Promise<{
		tables: number;
		totalRows?: number;
		error?: string;
	}> {
		try {
			return await this.executeWithRetry(async (client) => {
				// Get table count
				const tablesResult = await client.query(`
                    SELECT COUNT(*) as table_count
                    FROM information_schema.tables 
                    WHERE table_schema = 'public'
                `);

				let totalRows = 0;
				try {
					// Try to get row count from TelemetryTicks if it exists
					const rowsResult = await client.query(`
                        SELECT COUNT(*) as row_count
                        FROM TelemetryTicks
                    `);
					totalRows = Number.parseInt(rowsResult.rows[0]?.row_count || "0", 10);
				} catch (_e) {
					// Table might not exist, that's okay
					console.log("TelemetryTicks table not found or empty");
				}

				return {
					tables: Number.parseInt(tablesResult.rows[0]?.table_count || "0", 10),
					totalRows,
				};
			});
		} catch (error) {
			return {
				tables: 0,
				error: error instanceof Error ? error.message : String(error),
			};
		}
	}
}

// Create singleton instance
const questDBClient = new QuestDBClient();

// Export the main functions
export const getTelemetryData =
	questDBClient.getTelemetryData.bind(questDBClient);
export const getLaps = questDBClient.getLaps.bind(questDBClient);
export const getSessions = questDBClient.getSessions.bind(questDBClient);
export const checkQuestDBHealth = async () => {
	const result = await questDBClient.healthCheck();
	return result.healthy;
};

// Export additional utilities
export const getQuestDBStats = questDBClient.getStats.bind(questDBClient);
export const getQuestDBHealth = questDBClient.healthCheck.bind(questDBClient);
