package skills

import "math"

// ResampleMax puts a curve on a uniform grid of windows `window` milliseconds
// wide of the beatmap's own timeline (the curve's Time times speed): the first
// window is [0, window), the next [window, 2*window), and so on, `windows` of
// them. Each window gets one point holding the strongest point of the curve
// inside it, and its Time is the middle of the window on the curve's own
// timeline (the window's middle divided by speed), so multiplying by speed
// gives the middle of the window on the beatmap's timeline.
//
// Keeping the maximum, not an average, matches how the skills themselves are
// made (from the strongest strains) and keeps the highest point of the curve:
// the maximum of the result is the maximum of the curve whenever the curve is
// inside the grid. A window without points of the curve is 0, which is right
// for a break in the map or a stretch without streams; with carry it keeps the
// previous window's value instead, for a running total (memory). Points
// outside the grid (before 0 or past the last window) fall into the first and
// the last window.
//
// A window of 0 or less returns a copy of the curve (every point, no grid). A
// speed of 0 or less is taken as 1. The input is not modified.
func ResampleMax(points []StrainPoint, window int, speed float64, windows int, carry bool) []StrainPoint {
	if window <= 0 {
		return append([]StrainPoint(nil), points...)
	}
	if speed <= 0 {
		speed = 1
	}
	if windows <= 0 {
		return []StrainPoint{}
	}

	values := make([]float64, windows)
	seen := make([]bool, windows)
	for _, p := range points {
		k := int(math.Floor(float64(p.Time) * speed / float64(window)))
		k = min(max(k, 0), windows-1)
		if !seen[k] || p.Value > values[k] {
			values[k] = p.Value
			seen[k] = true
		}
	}

	out := make([]StrainPoint, windows)
	last := 0.0
	for k := range out {
		switch {
		case seen[k]:
			last = values[k]
		case !carry:
			values[k] = 0
		default:
			values[k] = last
		}
		out[k] = StrainPoint{
			Time:  int(math.Round((float64(k) + 0.5) * float64(window) / speed)),
			Value: values[k],
		}
	}
	return out
}

// WindowCount is the number of windows of `window` milliseconds (beatmap
// timeline) needed to hold the latest point of the given curves.
func WindowCount(window int, speed float64, curves ...[]StrainPoint) int {
	if window <= 0 {
		return 0
	}
	if speed <= 0 {
		speed = 1
	}

	last := 0.0
	for _, curve := range curves {
		if n := len(curve); n > 0 {
			last = math.Max(last, float64(curve[n-1].Time)*speed)
		}
	}
	return int(math.Floor(last/float64(window))) + 1
}
