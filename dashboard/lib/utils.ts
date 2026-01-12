import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export interface LapRow {
	lap_id: string;
}

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}
