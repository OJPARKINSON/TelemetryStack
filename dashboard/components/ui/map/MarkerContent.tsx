"use client";

import type { ReactNode } from "react";
import { createPortal } from "react-dom";

import { cn } from "../../../lib/utils";
import { useMarkerContext } from "./MapMarker";

type MarkerContentProps = {
	/** Custom marker content. Defaults to a blue dot if not provided */
	children?: ReactNode;
	/** Additional CSS classes for the marker container */
	className?: string;
};

function DefaultMarkerIcon() {
	return (
		<div className="relative h-4 w-4 rounded-full border-2 border-white bg-blue-500 shadow-lg" />
	);
}

export function MarkerContent({ children, className }: MarkerContentProps) {
	const { marker } = useMarkerContext();

	return createPortal(
		<div className={cn("relative cursor-pointer", className)}>
			{children || <DefaultMarkerIcon />}
		</div>,
		marker.getElement(),
	);
}
