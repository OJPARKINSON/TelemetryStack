import { useCallback, useEffect, useRef, useState } from "react";

export function useChartHover() {
	const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
	const hoverFrameRef = useRef<number | null>(null);

	const handleChartHover = useCallback((index: number | null) => {
		if (hoverFrameRef.current) {
			cancelAnimationFrame(hoverFrameRef.current);
		}
		hoverFrameRef.current = requestAnimationFrame(() => {
			setHoveredIndex(index);
		});
	}, []);

	const handleChartMouseLeave = useCallback(() => {
		if (hoverFrameRef.current) {
			cancelAnimationFrame(hoverFrameRef.current);
		}
	}, []);

	useEffect(() => {
		return () => {
			if (hoverFrameRef.current) {
				cancelAnimationFrame(hoverFrameRef.current);
			}
		};
	}, []);

	return { hoveredIndex, handleChartHover, handleChartMouseLeave };
}
