"use client";

import MapLibreGL, { type MarkerOptions } from "maplibre-gl";
import {
	createContext,
	type ReactNode,
	useContext,
	useEffect,
	useMemo,
} from "react";

import { useMap } from "./MapContext";

type MarkerContextValue = {
	marker: MapLibreGL.Marker;
	map: MapLibreGL.Map | null;
};

export const MarkerContext = createContext<MarkerContextValue | null>(null);

export function useMarkerContext() {
	const context = useContext(MarkerContext);
	if (!context) {
		throw new Error("Marker components must be used within MapMarker");
	}
	return context;
}

type MapMarkerProps = {
	/** Longitude coordinate for marker position */
	longitude: number;
	/** Latitude coordinate for marker position */
	latitude: number;
	/** Marker subcomponents (MarkerContent, MarkerPopup, MarkerTooltip, MarkerLabel) */
	children: ReactNode;
	/** Callback when marker is clicked */
	onClick?: (e: MouseEvent) => void;
	/** Callback when mouse enters marker */
	onMouseEnter?: (e: MouseEvent) => void;
	/** Callback when mouse leaves marker */
	onMouseLeave?: (e: MouseEvent) => void;
	/** Callback when marker drag starts (requires draggable: true) */
	onDragStart?: (lngLat: { lng: number; lat: number }) => void;
	/** Callback during marker drag (requires draggable: true) */
	onDrag?: (lngLat: { lng: number; lat: number }) => void;
	/** Callback when marker drag ends (requires draggable: true) */
	onDragEnd?: (lngLat: { lng: number; lat: number }) => void;
} & Omit<MarkerOptions, "element">;

export function MapMarker({
	longitude,
	latitude,
	children,
	onClick,
	onMouseEnter,
	onMouseLeave,
	onDragStart,
	onDrag,
	onDragEnd,
	draggable = false,
	...markerOptions
}: MapMarkerProps) {
	const { map } = useMap();

	const marker = useMemo(() => {
		const markerInstance = new MapLibreGL.Marker({
			...markerOptions,
			element: document.createElement("div"),
			draggable,
		}).setLngLat([longitude, latitude]);

		const handleClick = (e: MouseEvent) => onClick?.(e);
		const handleMouseEnter = (e: MouseEvent) => onMouseEnter?.(e);
		const handleMouseLeave = (e: MouseEvent) => onMouseLeave?.(e);

		markerInstance.getElement()?.addEventListener("click", handleClick);
		markerInstance
			.getElement()
			?.addEventListener("mouseenter", handleMouseEnter);
		markerInstance
			.getElement()
			?.addEventListener("mouseleave", handleMouseLeave);

		const handleDragStart = () => {
			const lngLat = markerInstance.getLngLat();
			onDragStart?.({ lng: lngLat.lng, lat: lngLat.lat });
		};
		const handleDrag = () => {
			const lngLat = markerInstance.getLngLat();
			onDrag?.({ lng: lngLat.lng, lat: lngLat.lat });
		};
		const handleDragEnd = () => {
			const lngLat = markerInstance.getLngLat();
			onDragEnd?.({ lng: lngLat.lng, lat: lngLat.lat });
		};

		markerInstance.on("dragstart", handleDragStart);
		markerInstance.on("drag", handleDrag);
		markerInstance.on("dragend", handleDragEnd);

		return markerInstance;

		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, []);

	useEffect(() => {
		if (!map) return;

		marker.addTo(map);

		return () => {
			marker.remove();
		};

		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [map]);

	if (
		marker.getLngLat().lng !== longitude ||
		marker.getLngLat().lat !== latitude
	) {
		marker.setLngLat([longitude, latitude]);
	}
	if (marker.isDraggable() !== draggable) {
		marker.setDraggable(draggable);
	}

	const currentOffset = marker.getOffset();
	const newOffset = markerOptions.offset ?? [0, 0];
	const [newOffsetX, newOffsetY] = Array.isArray(newOffset)
		? newOffset
		: [newOffset.x, newOffset.y];
	if (currentOffset.x !== newOffsetX || currentOffset.y !== newOffsetY) {
		marker.setOffset(newOffset);
	}

	if (marker.getRotation() !== markerOptions.rotation) {
		marker.setRotation(markerOptions.rotation ?? 0);
	}
	if (marker.getRotationAlignment() !== markerOptions.rotationAlignment) {
		marker.setRotationAlignment(markerOptions.rotationAlignment ?? "auto");
	}
	if (marker.getPitchAlignment() !== markerOptions.pitchAlignment) {
		marker.setPitchAlignment(markerOptions.pitchAlignment ?? "auto");
	}

	return (
		<MarkerContext.Provider value={{ marker, map }}>
			{children}
		</MarkerContext.Provider>
	);
}
