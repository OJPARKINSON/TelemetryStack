import { createHash } from "crypto";
import { type NextRequest, NextResponse } from "next/server";

const TILE_SOURCES = {
	dark: "https://cartodb-basemaps-a.global.ssl.fastly.net/dark_all",
	light: "https://cartodb-basemaps-a.global.ssl.fastly.net/light_all",
	satellite:
		"https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile",
};

export async function GET(
	request: NextRequest,
	{ params }: { params: Promise<{ params: string[] }> },
) {
	try {
		const resolvedParams = await params;
		const [theme, z, x, y] = resolvedParams.params;

		// Validate parameters
		if (!theme || !z || !x || !y) {
			return NextResponse.json(
				{ error: "Invalid tile parameters" },
				{ status: 400 },
			);
		}

		// Validate theme
		if (!(theme in TILE_SOURCES)) {
			return NextResponse.json({ error: "Invalid theme" }, { status: 400 });
		}

		// Validate numeric parameters
		const zNum = parseInt(z);
		const xNum = parseInt(x);
		const yNum = parseInt(y);

		if (isNaN(zNum) || isNaN(xNum) || isNaN(yNum)) {
			return NextResponse.json(
				{ error: "Invalid tile coordinates" },
				{ status: 400 },
			);
		}

		// Construct tile URL
		const baseUrl = TILE_SOURCES[theme as keyof typeof TILE_SOURCES];
		const tileUrl =
			theme === "satellite"
				? `${baseUrl}/${z}/${y}/${x}`
				: `${baseUrl}/${z}/${x}/${y}.png`;

		// Generate ETag for this tile based on URL (tiles never change)
		const etag = `"${createHash("md5").update(tileUrl).digest("hex")}"`;

		// Check if client has cached version
		const clientETag = request.headers.get("if-none-match");
		if (clientETag === etag) {
			return new NextResponse(null, {
				status: 304,
				headers: {
					ETag: etag,
					"Cache-Control": "public, max-age=86400, immutable",
				},
			});
		}

		// Fetch the tile with caching headers
		const response = await fetch(tileUrl, {
			headers: {
				"User-Agent": "IRacing-Display/1.0",
				Referer: request.headers.get("referer") || "",
				"Accept-Encoding": "gzip, deflate, br",
			},
		});

		if (!response.ok) {
			return NextResponse.json(
				{ error: "Failed to fetch tile" },
				{ status: response.status },
			);
		}

		const tileData = await response.arrayBuffer();
		const contentType = response.headers.get("content-type") || "image/png";

		// Return the tile with aggressive caching headers
		return new NextResponse(tileData, {
			status: 200,
			headers: {
				"Content-Type": contentType,
				"Cache-Control": "public, max-age=86400, s-maxage=86400, immutable",
				ETag: etag,
				"Last-Modified": new Date().toUTCString(),
				"Cross-Origin-Resource-Policy": "cross-origin",
				"Access-Control-Allow-Origin": "*",
				Vary: "Accept-Encoding",
			},
		});
	} catch (error) {
		console.error("Tile proxy error:", error);
		return NextResponse.json(
			{ error: "Internal server error" },
			{ status: 500 },
		);
	}
}

export async function OPTIONS() {
	return new NextResponse(null, {
		status: 200,
		headers: {
			"Access-Control-Allow-Origin": "*",
			"Access-Control-Allow-Methods": "GET, OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type",
		},
	});
}
