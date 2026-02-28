export function formatTime(totalSeconds: number | undefined) {
	if (!totalSeconds) return "--";
	const minutes = Math.floor(totalSeconds / 60);
	const remainingSeconds = totalSeconds % 60;

	const seconds = Math.floor(remainingSeconds);
	const milliseconds = Math.round((remainingSeconds % 1) * 1000);

	const padTo2Digits = (num: number) => {
		return num.toString().padStart(2, "0");
	};

	const paddedMinutes = padTo2Digits(minutes);
	const paddedSeconds = padTo2Digits(seconds);
	const paddedMilliseconds = milliseconds > 0 ? `.${milliseconds}` : "";

	return `${paddedMinutes}:${paddedSeconds}${paddedMilliseconds}`;
}
