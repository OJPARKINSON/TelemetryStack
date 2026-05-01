import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// https://vitejs.dev/config/
export default defineConfig({
	base: process.env.VITE_BASE_PATH || "/",
	plugins: [
		tailwindcss(),
		tanstackRouter({ target: "react", autoCodeSplitting: true }),
		react(),
	],
	server: {
		cors: false,
		proxy: {
			"/api": {
				target: process.env.API_URL ?? "http://omarchy.tail95aa0e.ts.net:8010/",
				changeOrigin: true,
				secure: false,
			},
		},
	},
});
