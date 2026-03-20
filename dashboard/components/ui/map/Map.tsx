"use client";

import MapLibreGL from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";
import { useTheme } from "next-themes";
import {
	forwardRef,
	type ReactNode,
	useCallback,
	useEffect,
	useImperativeHandle,
	useMemo,
	useRef,
	useState,
} from "react";

import { MapContext } from "./MapContext";

export type MapStyleOption = string | MapLibreGL.StyleSpecification;

type MapProps = {
	children?: ReactNode;
	/** Custom map styles for light and dark themes. Overrides the default Carto styles. */
	styles?: {
		light?: MapStyleOption;
		dark?: MapStyleOption;
	};
	/** Map projection type. Use `{ type: "globe" }` for 3D globe view. */
	projection?: MapLibreGL.ProjectionSpecification;
} & Omit<MapLibreGL.MapOptions, "container" | "style">;

export type MapRef = MapLibreGL.Map;

const defaultStyles = {
	dark: "https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json",
	light: "https://basemaps.cartocdn.com/gl/positron-gl-style/style.json",
};

const DefaultLoader = () => (
	<div className="absolute inset-0 flex items-center justify-center">
		<div className="flex gap-1">
			<span className="size-1.5 animate-pulse rounded-full bg-muted-foreground/60" />
			<span className="size-1.5 animate-pulse rounded-full bg-muted-foreground/60 [animation-delay:150ms]" />
			<span className="size-1.5 animate-pulse rounded-full bg-muted-foreground/60 [animation-delay:300ms]" />
		</div>
	</div>
);

export const NewMap = forwardRef<MapRef, MapProps>(function MapFunc(
	{ children, styles, projection, ...props },
	ref,
) {
	const containerRef = useRef<HTMLDivElement>(null);
	const [mapInstance, setMapInstance] = useState<MapLibreGL.Map | null>(null);
	const [isLoaded, setIsLoaded] = useState(false);
	const [isStyleLoaded, setIsStyleLoaded] = useState(false);
	const { resolvedTheme } = useTheme();
	const currentStyleRef = useRef<MapStyleOption | null>(null);
	const styleTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

	const mapStyles = useMemo(
		() => ({
			dark: styles?.dark ?? defaultStyles.dark,
			light: styles?.light ?? defaultStyles.light,
		}),
		[styles],
	);

	useImperativeHandle(ref, () => mapInstance as MapLibreGL.Map, [mapInstance]);

	const clearStyleTimeout = useCallback(() => {
		if (styleTimeoutRef.current) {
			clearTimeout(styleTimeoutRef.current);
			styleTimeoutRef.current = null;
		}
	}, []);

	useEffect(() => {
		if (!containerRef.current) return;

		const initialStyle =
			resolvedTheme === "dark" ? mapStyles.dark : mapStyles.light;
		currentStyleRef.current = initialStyle;

		const map = new MapLibreGL.Map({
			container: containerRef.current,
			style: initialStyle,
			renderWorldCopies: false,
			attributionControl: {
				compact: true,
			},
			...props,
		});

		const styleDataHandler = () => {
			clearStyleTimeout();
			// Delay to ensure style is fully processed before allowing layer operations
			// This is a workaround to avoid race conditions with the style loading
			styleTimeoutRef.current = setTimeout(() => {
				setIsStyleLoaded(true);
				if (projection) {
					map.setProjection(projection);
				}
			}, 150);
		};
		const loadHandler = () => setIsLoaded(true);

		map.on("load", loadHandler);
		map.on("styledata", styleDataHandler);
		setMapInstance(map);

		return () => {
			clearStyleTimeout();
			map.off("load", loadHandler);
			map.off("styledata", styleDataHandler);
			map.remove();
			setIsLoaded(false);
			setIsStyleLoaded(false);
			setMapInstance(null);
		};
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [
		clearStyleTimeout,
		resolvedTheme,
		mapStyles.dark,
		mapStyles.light,
		projection,
	]);

	useEffect(() => {
		if (!mapInstance || !resolvedTheme) return;

		const newStyle =
			resolvedTheme === "dark" ? mapStyles.dark : mapStyles.light;

		if (currentStyleRef.current === newStyle) return;

		clearStyleTimeout();
		currentStyleRef.current = newStyle;
		setIsStyleLoaded(false);

		mapInstance.setStyle(newStyle, { diff: true });
	}, [mapInstance, resolvedTheme, mapStyles, clearStyleTimeout]);

	const isLoading = !isLoaded || !isStyleLoaded;

	const contextValue = useMemo(
		() => ({
			map: mapInstance,
			isLoaded: isLoaded && isStyleLoaded,
		}),
		[mapInstance, isLoaded, isStyleLoaded],
	);

	return (
		<MapContext.Provider value={contextValue}>
			<div ref={containerRef} className="relative h-full w-full">
				{isLoading && <DefaultLoader />}
				{/* SSR-safe: children render only when map is loaded on client */}
				{mapInstance && children}
			</div>
		</MapContext.Provider>
	);
});
