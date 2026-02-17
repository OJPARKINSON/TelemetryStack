"use client";

import MapLibreGL, { type PopupOptions } from "maplibre-gl";
import { X } from "lucide-react";
import { type ReactNode, useEffect, useMemo, useRef } from "react";
import { createPortal } from "react-dom";

import { cn } from "../../../lib/utils";
import { useMarkerContext } from "./MapMarker";

type MarkerPopupProps = {
	/** Popup content */
	children: ReactNode;
	/** Additional CSS classes for the popup container */
	className?: string;
	/** Show a close button in the popup (default: false) */
	closeButton?: boolean;
} & Omit<PopupOptions, "className" | "closeButton">;

export function MarkerPopup({
	children,
	className,
	closeButton = false,
	...popupOptions
}: MarkerPopupProps) {
	const { marker, map } = useMarkerContext();
	const container = useMemo(() => document.createElement("div"), []);
	const prevPopupOptions = useRef(popupOptions);

	const popup = useMemo(() => {
		const popupInstance = new MapLibreGL.Popup({
			offset: 16,
			...popupOptions,
			closeButton: false,
		})
			.setMaxWidth("none")
			.setDOMContent(container);

		return popupInstance;
		// eslint-disable-next-line react-hooks/exhaustive-deps
		// biome-ignore lint/correctness/useExhaustiveDependencies: <explanation>
	}, []);

	useEffect(() => {
		if (!map) return;

		popup.setDOMContent(container);
		marker.setPopup(popup);

		return () => {
			marker.setPopup(null);
		};
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [map, container, marker.setPopup, popup]);

	if (popup.isOpen()) {
		const prev = prevPopupOptions.current;

		if (prev.offset !== popupOptions.offset) {
			popup.setOffset(popupOptions.offset ?? 16);
		}
		if (prev.maxWidth !== popupOptions.maxWidth && popupOptions.maxWidth) {
			popup.setMaxWidth(popupOptions.maxWidth ?? "none");
		}

		prevPopupOptions.current = popupOptions;
	}

	const handleClose = () => popup.remove();

	return createPortal(
		<div
			className={cn(
				"fade-in-0 zoom-in-95 relative animate-in rounded-md border bg-popover p-3 text-popover-foreground shadow-md",
				className,
			)}
		>
			{closeButton && (
				<button
					type="button"
					onClick={handleClose}
					className="absolute top-1 right-1 z-10 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
					aria-label="Close popup"
				>
					<X className="h-4 w-4" />
					<span className="sr-only">Close</span>
				</button>
			)}
			{children}
		</div>,
		container,
	);
}
