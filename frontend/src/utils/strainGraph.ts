import type { StrainPoint } from '@/types/strains'

/**
 * Cuts a curve into runs of points that are drawn as one line. The backend
 * thins curves to the strongest point per window and leaves the windows of a
 * break empty, so a gap bigger than `maxGap` ms means "no notes here" and the
 * line must not be drawn across it.
 */
export function splitSegments(points: StrainPoint[], maxGap: number): StrainPoint[][] {
  const segments: StrainPoint[][] = []
  let current: StrainPoint[] = []

  for (const p of points) {
    const last = current[current.length - 1]
    if (last && p.time - last.time > maxGap) {
      segments.push(current)
      current = []
    }
    current.push(p)
  }
  if (current.length > 0) segments.push(current)

  return segments
}

/**
 * Value of a curve at `time`: the value of the last point at or before it, if
 * that point is not further than `maxGap` ms away (otherwise there is nothing
 * playing there and the answer is null).
 */
export function valueAt(points: StrainPoint[], time: number, maxGap: number): number | null {
  let lo = 0
  let hi = points.length - 1
  let found = -1

  while (lo <= hi) {
    const mid = (lo + hi) >> 1
    if (points[mid].time <= time) {
      found = mid
      lo = mid + 1
    } else {
      hi = mid - 1
    }
  }

  // Before the first point: the first one holds (a window's point is in its middle).
  if (found < 0) return points.length > 0 && points[0].time - time <= maxGap ? points[0].value : null
  if (time - points[found].time > maxGap) return null
  return points[found].value
}

/** A "nice" tick step (1, 2, 5 × 10^n) giving about `count` ticks over `range`. */
export function niceStep(range: number, count: number): number {
  if (range <= 0 || count <= 0) return 1
  const raw = range / count
  const pow = Math.pow(10, Math.floor(Math.log10(raw)))
  const base = raw / pow
  return (base >= 5 ? 5 : base >= 2 ? 2 : 1) * pow
}

/** m:ss of milliseconds. */
export function formatClock(ms: number): string {
  const total = Math.max(0, Math.round(ms / 1000))
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`
}

/**
 * Turns the points of windows (each in the middle of a window `width` ms wide)
 * into a step line, so that every window is drawn as a plateau of its value
 * and the line does not slope between windows. For curves that are on or off
 * (tenacity: a stream, then none), where a slope would show something that is
 * not there.
 */
export function toSteps<T extends { time: number; value: number }>(points: T[], width: number): T[] {
  const half = width / 2
  const out: T[] = []
  for (const p of points) {
    out.push({ ...p, time: p.time - half }, { ...p, time: p.time + half })
  }
  return out
}
