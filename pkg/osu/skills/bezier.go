package skills

import "github.com/compico/go-osu/pkg/vector2d"

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
	var scratch [16]vector2d.Vector2dd
	var points []vector2d.Vector2dd
	if len(b.points) <= len(scratch) {
		points = scratch[:len(b.points)]
	} else {
		points = make([]vector2d.Vector2dd, len(b.points))
	}
	for i := 0; i < b.curveCount; i++ {
		t := float64(i) / float64(b.curveCount-1)
		copy(points, b.points)
		b.curvePoints[i] = b.pointAt(t, points)
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

// PointAt evaluates a point at parameter t in [0, 1].
func (b *Bezier) PointAt(t float64) vector2d.Vector2dd {
	var scratch [16]vector2d.Vector2dd
	var points []vector2d.Vector2dd
	if len(b.points) <= len(scratch) {
		points = scratch[:len(b.points)]
	} else {
		points = make([]vector2d.Vector2dd, len(b.points))
	}
	copy(points, b.points)
	return b.pointAt(t, points)
}

// pointAt uses de Casteljau's algorithm, which avoids the repeated powers in
// the expanded Bernstein sum while evaluating the same Bezier curve.
func (b *Bezier) pointAt(t float64, points []vector2d.Vector2dd) vector2d.Vector2dd {
	if len(points) == 0 {
		return vector2d.Vector2dd{}
	}
	for level := len(points) - 1; level > 0; level-- {
		for i := 0; i < level; i++ {
			points[i].X = lerp(points[i].X, points[i+1].X, t)
			points[i].Y = lerp(points[i].Y, points[i+1].Y, t)
		}
	}
	return points[0]
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
