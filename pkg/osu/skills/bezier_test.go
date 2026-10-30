package skills

import (
	"math"
	"testing"

	"github.com/compico/go-osu/pkg/vector2d"
)

func TestBezierPointAtMatchesBernsteinEvaluation(t *testing.T) {
	curves := [][]vector2d.Vector2dd{
		{
			vector2d.New[float64](0, 0),
			vector2d.New[float64](512, 384),
		},
		{
			vector2d.New[float64](0, 0),
			vector2d.New[float64](512, 384),
			vector2d.New[float64](64, 448),
		},
		{
			vector2d.New[float64](256, 384),
			vector2d.New[float64](64, 64),
			vector2d.New[float64](448, 64),
			vector2d.New[float64](256, 384),
		},
		{
			vector2d.New[float64](0, 256),
			vector2d.New[float64](512, 256),
			vector2d.New[float64](0, 256),
			vector2d.New[float64](512, 256),
			vector2d.New[float64](256, 0),
			vector2d.New[float64](256, 512),
			vector2d.New[float64](0, 0),
			vector2d.New[float64](512, 512),
			vector2d.New[float64](256, 256),
		},
	}
	tValues := []float64{0, 1e-6, 0.01, 0.1, 0.25, 0.5, 0.75, 0.9, 0.99, 1 - 1e-6, 1}

	for curveIndex, points := range curves {
		bezier := NewBezier(points)
		for _, tt := range tValues {
			got := bezier.PointAt(tt)
			want := referenceBezierPointAt(points, tt)
			if math.Abs(got.X-want.X) > 1e-9 || math.Abs(got.Y-want.Y) > 1e-9 {
				t.Errorf("curve %d at t=%g: got (%g, %g), Bernstein got (%g, %g)",
					curveIndex, tt, got.X, got.Y, want.X, want.Y)
			}
		}
	}
}

func referenceBezierPointAt(points []vector2d.Vector2dd, t float64) vector2d.Vector2dd {
	var result vector2d.Vector2dd
	n := len(points) - 1
	for i, point := range points {
		weight := referenceBinomialCoefficient(n, i) * math.Pow(t, float64(i)) * math.Pow(1-t, float64(n-i))
		result.X += point.X * weight
		result.Y += point.Y * weight
	}
	return result
}

func referenceBinomialCoefficient(n, k int) float64 {
	if k < 0 || k > n {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}
	if k > n/2 {
		k = n - k
	}
	result := 1.0
	for i := 1; i <= k; i++ {
		result = result * float64(n-i+1) / float64(i)
	}
	return result
}
