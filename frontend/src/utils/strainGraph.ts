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

  if (found < 0 || time - points[found].time > maxGap) return null
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
