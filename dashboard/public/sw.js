// Service Worker for aggressive tile caching ONLY
const CACHE_NAME = "iracing-tiles-v2";
const TILE_CACHE_NAME = "map-tiles-v2";

// Tile URL patterns to cache aggressively (ONLY tiles)
const TILE_PATTERNS = [
	/\/api\/tiles\//,
	/\/dashboard\/api\/tiles\//,
	/cartodb-basemaps/,
	/arcgisonline\.com/,
];

// Patterns to NEVER cache (let Next.js handle these)
const NEVER_CACHE_PATTERNS = [
	/\/_next\//,
	/\/static\//,
	/\.js$/,
	/\.css$/,
	/\.woff2?$/,
	/\.ttf$/,
	/\.otf$/,
	/\.eot$/,
	/\/api\/(?!tiles)/,
	/\/favicon/,
	/\/manifest/,
];

self.addEventListener("install", (_event) => {
	// Skip waiting to activate immediately
	self.skipWaiting();
});

self.addEventListener("activate", (event) => {
	event.waitUntil(
		caches.keys().then((cacheNames) => {
			return Promise.all(
				cacheNames.map((cacheName) => {
					// Delete old caches
					if (cacheName !== CACHE_NAME && cacheName !== TILE_CACHE_NAME) {
						return caches.delete(cacheName);
					}
					return false;
				}),
			);
		}),
	);
	// Take control of all clients immediately
	return self.clients.claim();
});

self.addEventListener("fetch", (event) => {
	try {
		const url = event.request.url;

		// Skip chrome-extension and other non-http requests
		if (!url.startsWith("http")) {
			return;
		}

		// NEVER cache Next.js assets - let Next.js handle them
		if (NEVER_CACHE_PATTERNS.some((pattern) => pattern.test(url))) {
			return; // Let the request go through normally
		}

		// ONLY handle tile requests with aggressive caching
		if (TILE_PATTERNS.some((pattern) => pattern.test(url))) {
			event.respondWith(handleTileRequest(event.request));
			return;
		}

		// For all other requests, don't interfere - let them go through normally
		return;
	} catch (error) {
		console.warn("Service Worker fetch handler error:", error);
		// Don't respond to the event if there's an error - let it pass through
		return;
	}
});

async function handleTileRequest(request) {
	try {
		const cache = await caches.open(TILE_CACHE_NAME);

		// Check cache first
		let response = await cache.match(request);

		if (response) {
			// Return cached tile immediately
			return response;
		}

		// Add timeout to prevent hanging requests
		const controller = new AbortController();
		const timeoutId = setTimeout(() => controller.abort(), 10000); // 10 second timeout

		try {
			// Fetch from network with timeout
			response = await fetch(request, {
				signal: controller.signal,
				mode: "cors",
				credentials: "omit",
			});
			clearTimeout(timeoutId);

			// Cache successful responses for 1 year
			if (response.ok && response.status === 200) {
				// Clone response before caching
				const responseToCache = response.clone();

				// Ensure proper MIME type for tiles
				const contentType =
					responseToCache.headers.get("content-type") || "image/png";

				// Add custom headers for aggressive caching
				const headers = new Headers();
				headers.set("Content-Type", contentType);
				headers.set("Cache-Control", "public, max-age=86400, immutable");
				headers.set("Cross-Origin-Resource-Policy", "cross-origin");
				headers.set("Access-Control-Allow-Origin", "*");
				headers.set("SW-Cached", "true");

				const cachedResponse = new Response(responseToCache.body, {
					status: responseToCache.status,
					statusText: responseToCache.statusText,
					headers: headers,
				});

				// Cache the tile
				cache.put(request, cachedResponse.clone());

				return cachedResponse;
			}

			return response;
		} catch (fetchError) {
			clearTimeout(timeoutId);
			throw fetchError;
		}
	} catch (error) {
		console.warn("Tile fetch error:", error.name, error.message);

		// Try to return cached version as fallback
		try {
			const cache = await caches.open(TILE_CACHE_NAME);
			const cachedResponse = await cache.match(request);
			if (cachedResponse) {
				return cachedResponse;
			}
		} catch (cacheError) {
			console.warn("Cache access error:", cacheError);
		}

		// Return a simple error response without causing more errors
		return new Response("Tile not available", {
			status: 503,
			statusText: "Service Unavailable",
			headers: {
				"Content-Type": "text/plain",
				"Cache-Control": "no-cache",
			},
		});
	}
}

// Handle tile cache warming
self.addEventListener("message", (event) => {
	if (event.data && event.data.type === "WARM_TILE_CACHE") {
		const { tiles } = event.data;
		warmTileCache(tiles);
	}
});

async function warmTileCache(tileUrls) {
	try {
		const cache = await caches.open(TILE_CACHE_NAME);

		// Limit concurrent requests to prevent overwhelming the server
		const batchSize = 5;
		const batches = [];

		for (let i = 0; i < tileUrls.length; i += batchSize) {
			batches.push(tileUrls.slice(i, i + batchSize));
		}

		// Process batches sequentially
		for (const batch of batches) {
			const promises = batch.map(async (url) => {
				try {
					const controller = new AbortController();
					const timeoutId = setTimeout(() => controller.abort(), 5000); // 5 second timeout for warming

					const response = await fetch(url, {
						signal: controller.signal,
						mode: "cors",
						credentials: "omit",
					});
					clearTimeout(timeoutId);

					if (response.ok) {
						await cache.put(url, response.clone());
					}
				} catch (error) {
					// Silently ignore warming errors to prevent console spam
					if (error.name !== "AbortError") {
						console.warn("Failed to warm cache for tile:", url, error.name);
					}
				}
			});

			await Promise.allSettled(promises);

			// Small delay between batches to prevent overwhelming
			if (batches.indexOf(batch) < batches.length - 1) {
				await new Promise((resolve) => setTimeout(resolve, 100));
			}
		}
	} catch (error) {
		console.warn("Tile cache warming failed:", error);
	}
}
