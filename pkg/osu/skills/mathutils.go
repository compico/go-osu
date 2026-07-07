package skills

import (
	"math"
	"sort"

	"github.com/compico/go-osu/pkg/vector2d"
)

// degToRad ports the DegToRad macro from utils.h.
func degToRad(deg float64) float64 {
	return deg * math.Pi / 180.0
}

// clampVal ports utils.h's BOUND/clamp usage for float64. It's a thin
// wrapper around the generic vector2d.Clamp so callers in this package
// don't need to repeat the type parameter everywhere.
func clampVal(value, low, high float64) float64 {
	return vector2d.Clamp(value, low, high)
}

// lerp ports utils.cpp's lerp: linear interpolation of a and b at t.
func lerp(a, b, t float64) float64 {
	return a*(1-t) + b*t
}

// smootherStep ports utils.cpp's smootherstep.
func smootherStep(x, start, end float64) float64 {
	x = clampVal((x-start)/(end-start), 0.0, 1.0)
	return x * x * x * (x*(6.0*x-15.0) + 10.0)
}

// reverseLerp ports utils.cpp's reverseLerp.
func reverseLerp(x, start, end float64) float64 {
	return clampVal((x-start)/(end-start), 0.0, 1.0)
}

// norm ports utils.cpp's norm: the p-norm of an n-dimensional vector.
func norm(p float64, values []float64) float64 {
	sum := 0.0
	for _, x := range values {
		sum += math.Pow(x, p)
	}
	return math.Pow(sum, 1.0/p)
}

// getValue ports utils.cpp's getValue: resolves a percent value between
// min and max (e.g. 50% of 100..200 is 150).
func getValue(min, max, percent float64) float64 {
	return math.Max(max, min) - (1.0-percent)*(math.Max(max, min)-math.Min(max, min))
}

// getWeightedValue2 ports utils.cpp's getWeightedValue2: combines values
// weighted by decay^index, favoring earlier (i.e. larger) values.
func getWeightedValue2(vals []float64, decay float64) float64 {
	result := 0.0
	for i, v := range vals {
		result += v * math.Pow(decay, float64(i))
	}
	return result
}

// getPeakVals ports utils.cpp's getPeakVals: collects local maxima (strictly
// greater than both neighbors) and returns them sorted descending.
//
// Note: the original iterates from index 1 to len(vals)-1 (exclusive),
// meaning a slice of length 0 or 1 never enters the loop condition
// (unsigned underflow makes `vals.size() - 1` huge, but the loop guard
// `i < vals.size() - 1` with i starting at 1 still never matches when the
// true count is 0 elements to check) — the Go version below intentionally
// mirrors "no peaks for len < 2" via a plain int comparison instead of
// relying on unsigned wraparound.
func getPeakVals(vals []float64) []float64 {
	var output []float64
	for i := 1; i+1 < len(vals); i++ {
		if vals[i] > vals[i-1] && vals[i] > vals[i+1] {
			output = append(output, vals[i])
		}
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(output)))
	return output
}

// arToMs ports utils.cpp's AR2ms: converts an AR value to the number of
// milliseconds an object is visible before it needs to be clicked.
func arToMs(ar float64) float64 {
	if ar <= 5.00 {
		return 1800 - (120 * ar)
	}
	return 1950 - (150 * ar)
}

// cs2px ports utils.cpp's CS2px: converts a CS value into a circle radius
// in osu!pixels (640x480 space).
func cs2px(cs float64) float64 {
	return 54.5 - (4.5 * cs)
}

// getDirAngle ports utils.cpp's GetDirAngle: the directional angle in
// degrees (-180 -> +180) at point b, going from a to b to c. Positive is
// counter-clockwise, negative is clockwise.
func getDirAngle(a, b, c vector2d.Vector2dd) float64 {
	ab := vector2d.Vector2dd{X: b.X - a.X, Y: b.Y - a.Y}
	cb := vector2d.Vector2dd{X: b.X - c.X, Y: b.Y - c.Y}

	dot := ab.X*cb.X + ab.Y*cb.Y
	cross := ab.X*cb.Y - ab.Y*cb.X

	alpha := math.Atan2(cross, dot)

	return alpha * 180.0 / math.Pi
}

// sign ports utils.h's sign template for float64.
func sign(x float64) int {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	default:
		return 0
	}
}
