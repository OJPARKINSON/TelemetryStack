export type ChartConfig = {
	title: string;
	dataKey: string;
	color: string;
	unit: string;
	yDomain: number[];
};

// Kept outside the lazily loaded chart bundle so the loading skeleton can
// render the same cards (titles, units, count) without pulling in recharts.
export const CHART_CONFIGS: ChartConfig[] = [
	{
		title: "Speed",
		dataKey: "Speed",
		color: "#f97316",
		unit: "km/h",
		yDomain: [0, 300],
	},
	{
		title: "Throttle",
		dataKey: "Throttle",
		color: "#22c55e",
		unit: "%",
		yDomain: [0, 100],
	},
	{
		title: "Brake",
		dataKey: "Brake",
		color: "#ef4444",
		unit: "%",
		yDomain: [0, 100],
	},
	{
		title: "Gear",
		dataKey: "Gear",
		color: "#8b5cf6",
		unit: "",
		yDomain: [0, 8],
	},
	{
		title: "RPM",
		dataKey: "RPM",
		color: "#06b6d4",
		unit: "",
		yDomain: [0, 8000],
	},
	{
		title: "Steering",
		dataKey: "SteeringWheelAngle",
		color: "#ec4899",
		unit: "deg",
		yDomain: [-3, 3],
	},
];
