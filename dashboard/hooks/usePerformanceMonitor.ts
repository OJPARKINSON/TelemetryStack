"use client";

import { useEffect, useRef } from "react";

interface PerformanceMetrics {
	renderTime: number;
	frameRate: number;
	memoryUsage?: number;
}

export function usePerformanceMonitor(componentName: string, enabled = true) {
	const lastFrameTimeRef = useRef<number>(performance.now());
	const frameCountRef = useRef<number>(0);
	const fpsRef = useRef<number>(60);
	const renderStartRef = useRef<number>(0);

	useEffect(() => {
		if (!enabled) return;

		renderStartRef.current = performance.now();

		return () => {
			const renderTime = performance.now() - renderStartRef.current;

			// Calculate FPS
			const currentTime = performance.now();
			const deltaTime = currentTime - lastFrameTimeRef.current;

			if (deltaTime >= 1000) {
				// Update every second
				fpsRef.current = Math.round((frameCountRef.current * 1000) / deltaTime);
				frameCountRef.current = 0;
				lastFrameTimeRef.current = currentTime;

				const metrics: PerformanceMetrics = {
					renderTime,
					frameRate: fpsRef.current,
				};

				// Add memory usage if available
				if ("memory" in performance) {
					metrics.memoryUsage = Math.round(
						(performance as any).memory.usedJSHeapSize / 1024 / 1024,
					);
				}

				console.log(`🚀 ${componentName} Performance:`, {
					renderTime: `${renderTime.toFixed(2)}ms`,
					fps: `${metrics.frameRate}fps`,
					memory: metrics.memoryUsage ? `${metrics.memoryUsage}MB` : "N/A",
				});

				// Warn about performance issues
				if (renderTime > 16) {
					console.warn(
						`⚠️ ${componentName} slow render: ${renderTime.toFixed(2)}ms (target: <16ms)`,
					);
				}

				if (metrics.frameRate < 30) {
					console.warn(
						`⚠️ ${componentName} low FPS: ${metrics.frameRate}fps (target: >30fps)`,
					);
				}
			}

			frameCountRef.current++;
		};
	}, [componentName, enabled]);

	return {
		getCurrentFPS: () => fpsRef.current,
		markRenderStart: () => {
			renderStartRef.current = performance.now();
		},
		markRenderEnd: () => performance.now() - renderStartRef.current,
	};
}
