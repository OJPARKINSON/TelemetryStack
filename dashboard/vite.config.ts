import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { compression } from "vite-plugin-compression2";
import { tanstackRouter } from "@tanstack/router-plugin/vite";

// https://vitejs.dev/config/
export default defineConfig({
	base: process.env.VITE_BASE_PATH || "/",
	worker: { format: "es" },
	plugins: [
		tailwindcss(),
		tanstackRouter({ target: "react", autoCodeSplitting: true }),
		react(),
		compression(),
	],
	server: {
		cors: false,
		proxy: {
			"/api": {
				target: process.env.API_URL ?? "http://localhost:8010/",
				changeOrigin: true,
				secure: false,
			},
		},
	},
});
