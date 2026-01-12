import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// https://vitejs.dev/config/
export default defineConfig({
	plugins: [
		tailwindcss(),
		tanstackRouter({ target: "react", autoCodeSplitting: true }),
		react(),
	],
	server: {
		cors: false,
		proxy: {
			"/api": {
				target: "http://localhost:8010/",
				changeOrigin: true,
				secure: false,
			},
		},
	},
});
