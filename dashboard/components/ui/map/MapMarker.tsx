"use client";

import MapLibreGL, { type MarkerOptions } from "maplibre-gl";
import {
	createContext,
	type ReactNode,
	use,
	useEffect,
	useMemo,
	useRef,
} from "react";

import { useMap } from "./MapContext";

type MarkerContextValue = {
	marker: MapLibreGL.Marker;
	map: MapLibreGL.Map | null;
};

const MarkerContext = createContext<MarkerContextValue | null>(null);

export function useMarkerContext() {
	const context = use(MarkerContext);
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
	/** Marker subcomponents (MarkerContent) */
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

	// Marker event handlers read the latest props via this ref so the
	// mount-once marker below never calls stale closures.
	const handlersRef = useRef({
		onClick,
		onMouseEnter,
		onMouseLeave,
		onDragStart,
		onDrag,
		onDragEnd,
	});
	handlersRef.current = {
		onClick,
		onMouseEnter,
		onMouseLeave,
		onDragStart,
		onDrag,
		onDragEnd,
	};

	// biome-ignore lint/correctness/useExhaustiveDependencies: marker is created once; position/options are synced during render below and handlers go through handlersRef
	// eslint-disable-next-line react-hooks/exhaustive-deps -- intentional mount-once instance; all reactive values are synced during render or read via handlersRef
	const marker = useMemo(() => {
		const markerInstance = new MapLibreGL.Marker({
			...markerOptions,
			element: document.createElement("div"),
			draggable,
		}).setLngLat([longitude, latitude]);

		const handleClick = (e: MouseEvent) => handlersRef.current.onClick?.(e);
		const handleMouseEnter = (e: MouseEvent) =>
			handlersRef.current.onMouseEnter?.(e);
		const handleMouseLeave = (e: MouseEvent) =>
			handlersRef.current.onMouseLeave?.(e);

		markerInstance.getElement()?.addEventListener("click", handleClick);
		markerInstance
			.getElement()
			?.addEventListener("mouseenter", handleMouseEnter);
		markerInstance
			.getElement()
			?.addEventListener("mouseleave", handleMouseLeave);

		const handleDragStart = () => {
			const lngLat = markerInstance.getLngLat();
			handlersRef.current.onDragStart?.({ lng: lngLat.lng, lat: lngLat.lat });
		};
		const handleDrag = () => {
			const lngLat = markerInstance.getLngLat();
			handlersRef.current.onDrag?.({ lng: lngLat.lng, lat: lngLat.lat });
		};
		const handleDragEnd = () => {
			const lngLat = markerInstance.getLngLat();
			handlersRef.current.onDragEnd?.({ lng: lngLat.lng, lat: lngLat.lat });
		};

		markerInstance.on("dragstart", handleDragStart);
		markerInstance.on("drag", handleDrag);
		markerInstance.on("dragend", handleDragEnd);

		return markerInstance;
	}, []);

	useEffect(() => {
		if (!map) return;

		marker.addTo(map);

		return () => {
			marker.remove();
		};

		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [map, marker.addTo, marker.remove]);

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

	const contextValue = useMemo(() => ({ marker, map }), [marker, map]);

	return (
		<MarkerContext.Provider value={contextValue}>
			{children}
		</MarkerContext.Provider>
	);
}
