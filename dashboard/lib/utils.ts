import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export interface LapRow {
	lap_id: string;
	[key: string]: any; // For any other properties
}

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}
