package skills

// MaxPerWindow thins a curve out: the time axis is cut into windows of the
// given length (in the curve's own milliseconds, the first starting at time 0)
// and only the strongest point of every window is kept, at its own time. The
// first of equal points wins. A window that has no points stays empty, so the
// result is sparse where the curve is (breaks in the map). Points stay ordered.
//
// Keeping the maximum, not an average, means the peaks the skills are made of
// survive: the highest point of the curve is always in the result, and the
// result never has a value the curve does not have. A window of 0 or less
// returns a copy of the curve.
//
// The input is not modified.
func MaxPerWindow(points []StrainPoint, window int) []StrainPoint {
	if window <= 0 {
		return append([]StrainPoint(nil), points...)
	}

	var out []StrainPoint
	lastWindow := 0
	for i, p := range points {
		w := floorDiv(p.Time, window)
		if i > 0 && w == lastWindow {
			if last := &out[len(out)-1]; p.Value > last.Value {
				*last = p
			}
			continue
		}
		out = append(out, p)
		lastWindow = w
	}
	return out
}

// floorDiv divides rounding towards negative infinity, so that negative times
// (hit objects before the start of the audio) get windows of the same length.
func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}
