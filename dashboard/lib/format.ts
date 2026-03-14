export function formatLapTime(seconds: number | undefined): string {
	if (!seconds || seconds <= 0) return "--:--.---";
	const mins = Math.floor(seconds / 60);
	const secs = seconds % 60;
	return `${mins}:${secs.toFixed(2).padStart(5, "0")}`;
}
