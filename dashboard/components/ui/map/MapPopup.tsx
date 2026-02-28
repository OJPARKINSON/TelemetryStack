"use client";

import MapLibreGL, { type PopupOptions } from "maplibre-gl";
import { X } from "lucide-react";
import { type ReactNode, useEffect, useMemo, useRef } from "react";
import { createPortal } from "react-dom";

import { cn } from "../../../lib/utils";
import { useMap } from "./MapContext";

type MapPopupProps = {
	/** Longitude coordinate for popup position */
	longitude: number;
	/** Latitude coordinate for popup position */
	latitude: number;
	/** Callback when popup is closed */
	onClose?: () => void;
	/** Popup content */
	children: ReactNode;
	/** Additional CSS classes for the popup container */
	className?: string;
	/** Show a close button in the popup (default: false) */
	closeButton?: boolean;
} & Omit<PopupOptions, "className" | "closeButton">;

export function MapPopup({
	longitude,
	latitude,
	onClose,
	children,
	className,
	closeButton = false,
	...popupOptions
}: MapPopupProps) {
	const { map } = useMap();
	const popupOptionsRef = useRef(popupOptions);
	const container = useMemo(() => document.createElement("div"), []);

	const popup = useMemo(() => {
		const popupInstance = new MapLibreGL.Popup({
			offset: 16,
			...popupOptions,
			closeButton: false,
		})
			.setMaxWidth("none")
			.setLngLat([longitude, latitude]);

		return popupInstance;
		// biome-ignore lint/correctness/useExhaustiveDependencies: <explanation>
	}, []);

	useEffect(() => {
		if (!map) return;

		const onCloseProp = () => onClose?.();
		popup.on("close", onCloseProp);

		popup.setDOMContent(container);
		popup.addTo(map);

		return () => {
			popup.off("close", onCloseProp);
			if (popup.isOpen()) {
				popup.remove();
			}
		};
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [map]);

	if (popup.isOpen()) {
		const prev = popupOptionsRef.current;

		if (
			popup.getLngLat().lng !== longitude ||
			popup.getLngLat().lat !== latitude
		) {
			popup.setLngLat([longitude, latitude]);
		}

		if (prev.offset !== popupOptions.offset) {
			popup.setOffset(popupOptions.offset ?? 16);
		}
		if (prev.maxWidth !== popupOptions.maxWidth && popupOptions.maxWidth) {
			popup.setMaxWidth(popupOptions.maxWidth ?? "none");
		}
		popupOptionsRef.current = popupOptions;
	}

	const handleClose = () => {
		popup.remove();
		onClose?.();
	};

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
