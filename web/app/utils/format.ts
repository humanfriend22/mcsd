export function formatUptime(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  const totalMinutes = Math.floor(seconds / 60)
  if (totalMinutes < 60) return `${totalMinutes}m`
  const totalHours = Math.floor(totalMinutes / 60)
  const remainingMinutes = totalMinutes % 60
  if (totalHours < 24) return remainingMinutes > 0 ? `${totalHours}h ${remainingMinutes}m` : `${totalHours}h`
  const totalDays = Math.floor(totalHours / 24)
  const remainingHours = totalHours % 24
  return remainingHours > 0 ? `${totalDays}d ${remainingHours}h` : `${totalDays}d`
}

// formatUptimeSince turns a server-sent "active since" timestamp into a live
// duration, computed against the browser's own clock.
export function formatUptimeSince(activeSince: string | null): string {
  if (!activeSince) return '—'
  const seconds = Math.floor((Date.now() - new Date(activeSince).getTime()) / 1000)
  return formatUptime(Math.max(seconds, 0))
}

export function formatMemoryMB(mb: number): string {
  return mb >= 1024 ? `${(mb / 1024).toFixed(1)} GB` : `${mb} MB`
}
