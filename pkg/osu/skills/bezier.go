package skills

import (
	"math"

	"github.com/compico/go-osu/pkg/vector2d"
)

type Bezier struct {
	points        []vector2d.Vector2dd
	curvePoints   []vector2d.Vector2dd
	curveDis      []float64
	curveCount    int
	totalDistance float64
}

// NewBezier creates a Bezier curve approximated via point densities
func NewBezier(points []vector2d.Vector2dd) *Bezier {
	b := &Bezier{
		points: points,
	}

	var approxLength float64 = 0
	for i := 0; i < len(points)-1; i++ {
		approxLength += points[i].DistanceFrom(points[i+1])
	}

	b.Init(approxLength)
	return b
}

func (b *Bezier) Init(approxLength float64) {
	b.curveCount = int(approxLength/4.0) + 2
	b.curvePoints = make([]vector2d.Vector2dd, b.curveCount)
	for i := 0; i < b.curveCount; i++ {
		t := float64(i) / float64(b.curveCount-1)
		b.curvePoints[i] = b.PointAt(t)
	}

	b.curveDis = make([]float64, b.curveCount)
	b.totalDistance = 0
	for i := 0; i < b.curveCount; i++ {
		if i == 0 {
			b.curveDis[i] = 0
		} else {
			b.curveDis[i] = b.curvePoints[i].DistanceFrom(b.curvePoints[i-1])
		}
		b.totalDistance += b.curveDis[i]
	}
}

// PointAt evaluates a point at parameter t in [0, 1] using Bernstein polynomials
func (b *Bezier) PointAt(t float64) vector2d.Vector2dd {
	var c vector2d.Vector2dd
	n := len(b.points) - 1
	for i := 0; i <= n; i++ {
		bern := bernstein(i, n, t)
		c.X += b.points[i].X * bern
		c.Y += b.points[i].Y * bern
	}
	return c
}

func bernstein(i, n int, t float64) float64 {
	return binomialCoefficient(n, i) * math.Pow(t, float64(i)) * math.Pow(1-t, float64(n-i))
}

func binomialCoefficient(n, k int) float64 {
	if k < 0 || k > n {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}
	if k > n/2 {
		k = n - k
	}
	res := 1.0
	for i := 1; i <= k; i++ {
		res = res * float64(n-i+1) / float64(i)
	}
	return res
}

func (b *Bezier) GetCurvePoints() []vector2d.Vector2dd {
	return b.curvePoints
}

func (b *Bezier) GetCurveDistances() []float64 {
	return b.curveDis
}

func (b *Bezier) GetCurvesCount() int {
	return b.curveCount
}

func (b *Bezier) GetTotalDistance() float64 {
	return b.totalDistance
}
