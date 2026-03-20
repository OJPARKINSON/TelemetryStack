import type VectorSource from "ol/source/Vector";
import { useRef } from "react";

export function useMapLayers() {
	const racingLineSourceRef = useRef<VectorSource | null>(null);
	const carPositionSourceRef = useRef<VectorSource | null>(null);
	const selectedMarkerSourceRef = useRef<VectorSource | null>(null);
	const hoverMarkerSourceRef = useRef<VectorSource | null>(null);

	return {
		racingLineSourceRef,
		carPositionSourceRef,
		selectedMarkerSourceRef,
		hoverMarkerSourceRef,
	};
}
