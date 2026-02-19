import { Container } from "@cloudflare/containers";

export class IracingAPI extends Container {
	defaultPort = 8080;
	sleepAfter = "30s";

	override onStart() {
		console.log("iRacing API container started");
	}

	override onStop() {
		console.log("iRacing API container stopped");
	}

	override onError(error: unknown) {
		console.error("iRacing API container error:", error);
	}
}

interface Env {
	IRACING_API: DurableObjectNamespace;
}

export default {
	async fetch(request: Request, env: Env): Promise<Response> {
		const container = env.IRACING_API.getByName("iracing-api");
		return await container.fetch(request);
	},
};
