"use client";

import MapLibreGL, { type PopupOptions } from "maplibre-gl";
import { type ReactNode, useEffect, useMemo, useRef } from "react";
import { createPortal } from "react-dom";

import { cn } from "../../../lib/utils";
import { useMarkerContext } from "./MapMarker";

type MarkerTooltipProps = {
	/** Tooltip content */
	children: ReactNode;
	/** Additional CSS classes for the tooltip container */
	className?: string;
} & Omit<PopupOptions, "className" | "closeButton" | "closeOnClick">;

export function MarkerTooltip({
	children,
	className,
	...popupOptions
}: MarkerTooltipProps) {
	const { marker, map } = useMarkerContext();
	const container = useMemo(() => document.createElement("div"), []);
	const prevTooltipOptions = useRef(popupOptions);

	const tooltip = useMemo(() => {
		const tooltipInstance = new MapLibreGL.Popup({
			offset: 16,
			...popupOptions,
			closeOnClick: true,
			closeButton: false,
		}).setMaxWidth("none");

		return tooltipInstance;
		// eslint-disable-next-line react-hooks/exhaustive-deps
		// biome-ignore lint/correctness/useExhaustiveDependencies: <explanation>
	}, []);

	useEffect(() => {
		if (!map) return;

		tooltip.setDOMContent(container);

		const handleMouseEnter = () => {
			tooltip.setLngLat(marker.getLngLat()).addTo(map);
		};
		const handleMouseLeave = () => tooltip.remove();

		marker.getElement()?.addEventListener("mouseenter", handleMouseEnter);
		marker.getElement()?.addEventListener("mouseleave", handleMouseLeave);

		return () => {
			marker.getElement()?.removeEventListener("mouseenter", handleMouseEnter);
			marker.getElement()?.removeEventListener("mouseleave", handleMouseLeave);
			tooltip.remove();
		};
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [map]);

	if (tooltip.isOpen()) {
		const prev = prevTooltipOptions.current;

		if (prev.offset !== popupOptions.offset) {
			tooltip.setOffset(popupOptions.offset ?? 16);
		}
		if (prev.maxWidth !== popupOptions.maxWidth && popupOptions.maxWidth) {
			tooltip.setMaxWidth(popupOptions.maxWidth ?? "none");
		}

		prevTooltipOptions.current = popupOptions;
	}

	return createPortal(
		<div
			className={cn(
				"fade-in-0 zoom-in-95 animate-in rounded-md bg-foreground px-2 py-1 text-background text-xs shadow-md",
				className,
			)}
		>
			{children}
		</div>,
		container,
	);
}
