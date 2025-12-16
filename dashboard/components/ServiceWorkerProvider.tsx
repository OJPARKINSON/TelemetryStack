"use client";

import { useEffect } from "react";
import {
	registerServiceWorker,
	unregisterServiceWorker,
} from "../lib/sw-register";

export default function ServiceWorkerProvider() {
	useEffect(() => {
		if (process.env.NODE_ENV === "development") {
			// In development, actively unregister any existing Service Workers
			unregisterServiceWorker();
			console.log("Development mode: Service Worker disabled and unregistered");
			return;
		}

		// Only register Service Worker in production
		registerServiceWorker();
	}, []);

	return null; // This component doesn't render anything
}
