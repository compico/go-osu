package skills

import (
	"math"
	"sort"

	"github.com/chewxy/math32"
	"github.com/compico/go-osu/pkg/vector2d"
)

// degToRad ports the DegToRad macro from utils.h.
func degToRad(deg float64) float64 {
	return deg * math.Pi / 180.0
}

// btwn ports utils.h's BTWN macro: is b within [a, c]?
func btwn(lss, val, gtr float64) bool {
	return (min(lss, gtr) <= val) && (val <= max(lss, gtr))
}

// getAngle ports reaction.cpp's GetAngle: the angle in radians (0..pi)
// between points a-b-c.
func getAngle(a, b, c vector2d.Vector2dd) float64 {
	return math.Abs(degToRad(getDirAngle(a, b, c)))
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
func getPeakVals(vals []float64) (output []float64) {
	output = make([]float64, 0)

	for i := 1; i < len(vals)-1; i++ {
		if vals[i] > vals[i-1] && vals[i] > vals[i+1] {
			output = append(output, vals[i])
		}
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(output)))
	return
}

// arToMs ports utils.cpp's AR2ms: converts an AR value to the number of
// milliseconds an object is visible before it needs to be clicked.
func arToMs(ar float64) int {
	if ar <= 5.00 {
		return int(1800 - (120 * ar))
	}
	return int(1950 - (150 * ar))
}

// cs2px ports utils.cpp's CS2px: converts a CS value into a circle radius
// in osu!pixels (640x480 space).
func cs2px(cs float64) int {
	return int(54.5 - (4.5 * cs))
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
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}

// erfInv вычисляет обратную функцию ошибки.
// Область определения: x ∈ (-1, 1).
func erfInv(x float32) float32 {
	// Особые случаи.
	if x <= -1 {
		if x == -1 {
			return float32(math.Inf(-1))
		}
		return float32(math.NaN())
	}
	if x >= 1 {
		if x == 1 {
			return float32(math.Inf(1))
		}
		return float32(math.NaN())
	}

	w := -math32.Log((1 - x) * (1 + x))

	var p float32

	if w < 5 {
		w -= 2.5

		p = 2.81022636e-08
		p = 3.43273939e-07 + p*w
		p = -3.5233877e-06 + p*w
		p = -4.39150654e-06 + p*w
		p = 2.18580870e-04 + p*w
		p = -1.25372503e-03 + p*w
		p = -4.17768164e-03 + p*w
		p = 2.46640727e-01 + p*w
		p = 1.50140941 + p*w
	} else {
		w = math32.Sqrt(w) - 3

		p = -2.00214257e-04
		p = 1.00950558e-04 + p*w
		p = 1.34934322e-03 + p*w
		p = -3.67342844e-03 + p*w
		p = 5.73950773e-03 + p*w
		p = -7.62246130e-03 + p*w
		p = 9.43887047e-03 + p*w
		p = 1.00167406 + p*w
		p = 2.83297682 + p*w
	}

	return p * x
}

// getValuePos finds the index of the closest preceding timing point offset
func getValuePos(offsets []float64, value float64) int {
	if len(offsets) == 0 {
		return -1
	}

	for i := 0; i < len(offsets)-1; i++ {
		if offsets[i+1] > value {
			return i
		}
	}

	return len(offsets) - 1
}
