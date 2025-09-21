/** @type {import('next').NextConfig} */
const nextConfig = {
	output: process.env.NODE_ENV === "production" ? "standalone" : undefined,
	serverExternalPackages: ["pg"],
	reactStrictMode: true,

	// Performance optimizations
	experimental: {
		optimizePackageImports: ["recharts"],
	},

	turbopack: {
		rules: {
			"*.svg": {
				loaders: ["@svgr/webpack"],
				as: "*.js",
			},
		},
	},

	compiler: {
		removeConsole:
			process.env.NODE_ENV === "production"
				? {
						exclude: ["error", "warn"],
					}
				: false,
	},

	basePath: process.env.NODE_ENV === "production" ? "/dashboard" : "",

	images: {
		formats: ["image/webp", "image/avif"],
		minimumCacheTTL: 86400,
		dangerouslyAllowSVG: true,
		contentSecurityPolicy: "default-src 'self'; script-src 'none'; sandbox;",
	},

	async headers() {
		return [
			{
				source: "/api/:path*",
				headers: [
					{ key: "Access-Control-Allow-Credentials", value: "true" },
					{ key: "Access-Control-Allow-Origin", value: "*" },
					{
						key: "Access-Control-Allow-Methods",
						value: "GET,OPTIONS,PATCH,DELETE,POST,PUT",
					},
					{
						key: "Access-Control-Allow-Headers",
						value:
							"X-CSRF-Token, X-Requested-With, Accept, Accept-Version, Content-Length, Content-MD5, Content-Type, Date, X-Api-Version",
					},
					// Cache API responses for 30 seconds
					{
						key: "Cache-Control",
						value: "public, s-maxage=30, stale-while-revalidate=60",
					},
				],
			},
			{
				source: "/dashboard/api/tiles/:path*",
				headers: [
					{ key: "Access-Control-Allow-Origin", value: "*" },
					{ key: "Access-Control-Allow-Methods", value: "GET" },
					{ key: "Cross-Origin-Resource-Policy", value: "cross-origin" },
					{ key: "Cross-Origin-Embedder-Policy", value: "unsafe-none" },
					// Aggressive caching for map tiles (1 year) - tiles never change
					{
						key: "Cache-Control",
						value: "public, max-age=86400, s-maxage=86400, immutable",
					},
					{ key: "Vary", value: "Accept-Encoding" },
				],
			},
			{
				source: "/dashboard/:path*-tiles/:path*",
				headers: [
					{ key: "Access-Control-Allow-Origin", value: "*" },
					{ key: "Access-Control-Allow-Methods", value: "GET" },
					{ key: "Cross-Origin-Resource-Policy", value: "cross-origin" },
					// Aggressive caching for map tiles (1 year) - tiles never change
					{
						key: "Cache-Control",
						value: "public, max-age=86400, s-maxage=86400, immutable",
					},
					{ key: "Vary", value: "Accept-Encoding" },
				],
			},
			{
				source: "/api/tiles/:path*",
				headers: [
					{ key: "Access-Control-Allow-Origin", value: "*" },
					{ key: "Access-Control-Allow-Methods", value: "GET" },
					{ key: "Cross-Origin-Resource-Policy", value: "cross-origin" },
					{ key: "Cross-Origin-Embedder-Policy", value: "unsafe-none" },
					// Aggressive caching for map tiles (1 year) - tiles never change
					{
						key: "Cache-Control",
						value: "public, max-age=86400, s-maxage=86400, immutable",
					},
					{ key: "Vary", value: "Accept-Encoding" },
				],
			},
			{
				source: "/:path*-tiles/:path*",
				headers: [
					{ key: "Access-Control-Allow-Origin", value: "*" },
					{ key: "Access-Control-Allow-Methods", value: "GET" },
					{ key: "Cross-Origin-Resource-Policy", value: "cross-origin" },
					// Aggressive caching for map tiles (1 year) - tiles never change
					{
						key: "Cache-Control",
						value: "public, max-age=86400, s-maxage=86400, immutable",
					},
					{ key: "Vary", value: "Accept-Encoding" },
				],
			},
			{
				source: "/_next/static/:path*",
				headers: [
					{
						key: "Cache-Control",
						value: "public, max-age=86400, immutable",
					},
				],
			},
			{
				source: "/static/:path*",
				headers: [
					{
						key: "Cache-Control",
						value: "public, max-age=86400, immutable",
					},
				],
			},
			// Security headers
			{
				source: "/(.*)",
				headers: [
					{
						key: "X-DNS-Prefetch-Control",
						value: "on",
					},
					{
						key: "X-Frame-Options",
						value: "DENY",
					},
					{
						key: "X-Content-Type-Options",
						value: "nosniff",
					},
					{
						key: "Referrer-Policy",
						value: "origin-when-cross-origin",
					},
					{
						key: "Cross-Origin-Embedder-Policy",
						value: "unsafe-none",
					},
					{
						key: "Cross-Origin-Opener-Policy",
						value: "same-origin-allow-popups",
					},
				],
			},
		];
	},

	// Map tile rewrites are now handled by Traefik for better performance
	async rewrites() {
		return [];
	},

	// Redirect configuration for performance
	async redirects() {
		return [
			{
				source: "/telemetry",
				destination: "/",
				permanent: false,
			},
		];
	},

	// PoweredByHeader removal for security and performance
	poweredByHeader: false,

	// Compression
	compress: true,

	// Generate ETags for better caching
	generateEtags: true,

	// Optimize page extensions
	pageExtensions: ["tsx", "ts", "jsx", "js"],

	// Environment variables for optimization
	env: {
		OPTIMIZE_IMAGES: "true",
		MINIMIZE_JS: process.env.NODE_ENV === "production" ? "true" : "false",
	},

	// Webpack optimizations
	webpack: (config, { isServer }) => {
		// Optimize bundle splitting
		if (!isServer) {
			config.optimization.splitChunks = {
				chunks: "all",
				cacheGroups: {
					ol: {
						name: "ol",
						test: /[\\/]node_modules[\\/]ol[\\/]/,
						priority: 30,
						chunks: "all",
					},
					recharts: {
						name: "recharts",
						test: /[\\/]node_modules[\\/]recharts[\\/]/,
						priority: 20,
						chunks: "all",
					},
					vendor: {
						name: "vendor",
						test: /[\\/]node_modules[\\/]/,
						priority: 10,
						chunks: "all",
					},
				},
			};
		}

		return config;
	},
};

export default nextConfig;
