// A tracked or focused span, as "2h 05m" / "45m". Minutes are padded only when
// hours are shown, so a column of durations lines up.
export function formatDuration(seconds: number): string {
	const total = Math.max(0, Math.round(seconds))
	const hours = Math.floor(total / 3600)
	const minutes = Math.floor((total % 3600) / 60)
	return hours > 0 ? `${hours}h ${minutes.toString().padStart(2, '0')}m` : `${minutes}m`
}
