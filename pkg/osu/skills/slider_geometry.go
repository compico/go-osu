package skills

import (
	"math"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

const (
	// sliderApproximationScale is the distance spacing used to approximate
	// the slider path with discrete points. This matches the value used in
	// the original osu! client and Kert's port.
	sliderApproximationScale = 0.5 // osu!pixels
)

// Path represents a continuous curve (a segment of a slider).
// It can be linear, Bezier, or a perfect circle arc.
type Path struct {
	points []vector2d.Vector2dd
	length float64
}

// NewPath creates a new Path from a slice of control points and its type.
// It pre-calculates the entire path into discrete LerpPoints using the
// standard approximation scale.
func NewPath(curveType osu.CurveType, controlPoints []vector2d.Vector2dd) *Path {
	if len(controlPoints) == 0 {
		return &Path{}
	}

	var pathPoints []vector2d.Vector2dd

	switch curveType {
	case osu.LinearCurve:
		pathPoints = createLinearPath(controlPoints)
	case osu.BezierCurve:
		pathPoints = createBezierPath(controlPoints)
	case osu.PerfectCurve:
		pathPoints = createPerfectPath(controlPoints)
	case osu.CatmullCurve:
		// Catmull curves are converted to Bezier curves in the original client.
		bezierControls := catmullToBezier(controlPoints)
		pathPoints = createBezierPath(bezierControls)
	default:
		// Fallback to linear if type is unknown
		pathPoints = createLinearPath(controlPoints)
	}

	// Calculate total length
	totalLength := 0.0
	for i := 1; i < len(pathPoints); i++ {
		totalLength += pathPoints[i-1].DistanceFrom(pathPoints[i])
	}

	return &Path{
		points: pathPoints,
		length: totalLength,
	}
}

// PointAt returns the point on the path at a given distance from the start.
// If the distance is greater than the path's length, it returns the last point.
func (p *Path) PointAt(distance float64) vector2d.Vector2dd {
	if len(p.points) == 0 {
		return vector2d.Vector2dd{}
	}
	if len(p.points) == 1 || distance <= 0 {
		return p.points[0]
	}
	if distance >= p.length {
		return p.points[len(p.points)-1]
	}

	// Walk through the path segments to find the correct one
	accumulatedLength := 0.0
	for i := 1; i < len(p.points); i++ {
		segmentStart := p.points[i-1]
		segmentEnd := p.points[i]
		segmentLength := segmentStart.DistanceFrom(segmentEnd)

		if accumulatedLength+segmentLength >= distance {
			// The point is on this segment
			t := (distance - accumulatedLength) / segmentLength
			return vector2d.Vector2dd{
				X: segmentStart.X + t*(segmentEnd.X-segmentStart.X),
				Y: segmentStart.Y + t*(segmentEnd.Y-segmentStart.Y),
			}
		}
		accumulatedLength += segmentLength
	}

	// Fallback
	return p.points[len(p.points)-1]
}

// Length returns the total length of the path.
func (p *Path) Length() float64 {
	return p.length
}

// --- Helper functions for creating paths ---

// createLinearPath creates a path for a linear slider segment.
// For a linear path, the control points are just the start and end.
func createLinearPath(points []vector2d.Vector2dd) []vector2d.Vector2dd {
	if len(points) < 2 {
		return points
	}

	start := points[0]
	end := points[len(points)-1]
	distance := start.DistanceFrom(end)

	numSteps := int(math.Ceil(distance / sliderApproximationScale))
	if numSteps < 2 {
		numSteps = 2
	}

	result := make([]vector2d.Vector2dd, numSteps)
	for i := 0; i < numSteps; i++ {
		t := float64(i) / float64(numSteps-1)
		result[i] = vector2d.Vector2dd{
			X: start.X + t*(end.X-start.X),
			Y: start.Y + t*(end.Y-start.Y),
		}
	}

	return result
}

// bezierPoint calculates a point on a Bezier curve at parameter t.
// Uses De Casteljau's algorithm recursively.
func bezierPoint(t float64, points []vector2d.Vector2dd) vector2d.Vector2dd {
	if len(points) == 1 {
		return points[0]
	}

	newPoints := make([]vector2d.Vector2dd, len(points)-1)
	for i := 0; i < len(points)-1; i++ {
		newPoints[i] = vector2d.Vector2dd{
			X: points[i].X + t*(points[i+1].X-points[i].X),
			Y: points[i].Y + t*(points[i+1].Y-points[i].Y),
		}
	}

	return bezierPoint(t, newPoints)
}

// createBezierPath approximates a Bezier curve with discrete points.
func createBezierPath(controlPoints []vector2d.Vector2dd) []vector2d.Vector2dd {
	if len(controlPoints) < 2 {
		return controlPoints
	}

	// Estimate the number of steps needed based on control point spread
	bboxMin := controlPoints[0]
	bboxMax := controlPoints[0]
	for _, p := range controlPoints {
		bboxMin.X = math.Min(bboxMin.X, p.X)
		bboxMin.Y = math.Min(bboxMin.Y, p.Y)
		bboxMax.X = math.Max(bboxMax.X, p.X)
		bboxMax.Y = math.Max(bboxMax.Y, p.Y)
	}
	dd := vector2d.Vector2dd{X: bboxMax.X - bboxMin.X, Y: bboxMax.Y - bboxMin.Y}
	bboxDiagonal := dd.Length()
	numSteps := int(math.Ceil(bboxDiagonal / sliderApproximationScale))
	if numSteps < 2 {
		numSteps = 2
	}

	result := make([]vector2d.Vector2dd, numSteps)
	for i := 0; i < numSteps; i++ {
		t := float64(i) / float64(numSteps-1)
		result[i] = bezierPoint(t, controlPoints)
	}

	return result
}

// calculateCircumcircle finds the center and radius of a circle passing through three points.
// Returns (center, radius, success). If points are collinear, success is false.
func calculateCircumcircle(a, b, c vector2d.Vector2dd) (vector2d.Vector2dd, float64, bool) {
	// Translate so that point A is at the origin
	b = b.Sub(a)
	c = c.Sub(a)

	d := 2 * (b.X*c.Y - b.Y*c.X)
	if math.Abs(d) < 1e-10 {
		// Points are collinear
		return vector2d.Vector2dd{}, 0, false
	}

	radiusSq := (b.X*b.X + b.Y*b.Y) * (c.X*c.X + c.Y*c.Y) * ((b.X-c.X)*(b.X-c.X) + (b.Y-c.Y)*(b.Y-c.Y)) / (d * d)
	if radiusSq < 0 {
		return vector2d.Vector2dd{}, 0, false
	}
	radius := math.Sqrt(radiusSq)

	center := vector2d.Vector2dd{
		X: (c.Y*(b.X*b.X+b.Y*b.Y) - b.Y*(c.X*c.X+c.Y*c.Y)) / d,
		Y: (b.X*(c.X*c.X+c.Y*c.Y) - c.X*(b.X*b.X+b.Y*b.Y)) / d,
	}

	// Translate back
	center = center.Add(a)

	return center, radius, true
}

// createPerfectPath creates a path for a perfect curve (circular arc).
func createPerfectPath(points []vector2d.Vector2dd) []vector2d.Vector2dd {
	if len(points) < 3 {
		// Not enough points for a circle, fall back to linear
		return createLinearPath(points)
	}

	a := points[0]
	b := points[len(points)/2] // Use the middle control point
	c := points[len(points)-1]

	center, radius, ok := calculateCircumcircle(a, b, c)
	if !ok {
		// Fall back to linear if points are collinear
		return createLinearPath(points)
	}

	// Calculate angles from the center to each point
	startAngle := math.Atan2(a.Y-center.Y, a.X-center.X)
	endAngle := math.Atan2(c.Y-center.Y, c.X-center.X)

	// Determine the direction of the arc (shortest path)
	angleDiff := endAngle - startAngle
	if angleDiff > math.Pi {
		angleDiff -= 2 * math.Pi
	} else if angleDiff < -math.Pi {
		angleDiff += 2 * math.Pi
	}

	arcLength := math.Abs(angleDiff) * radius
	numSteps := int(math.Ceil(arcLength / sliderApproximationScale))
	if numSteps < 2 {
		numSteps = 2
	}

	result := make([]vector2d.Vector2dd, numSteps)
	for i := 0; i < numSteps; i++ {
		t := float64(i) / float64(numSteps-1)
		currentAngle := startAngle + t*angleDiff
		result[i] = vector2d.Vector2dd{
			X: center.X + radius*math.Cos(currentAngle),
			Y: center.Y + radius*math.Sin(currentAngle),
		}
	}

	return result
}

// catmullToBezier converts a sequence of Catmull-Rom control points into
// a sequence of cubic Bezier control points.
// This is a standard conversion formula.
func catmullToBezier(catmullPoints []vector2d.Vector2dd) []vector2d.Vector2dd {
	if len(catmullPoints) < 2 {
		return catmullPoints
	}

	// Add phantom points at the start and end for a closed conversion
	sub := catmullPoints[0].Sub(catmullPoints[1])
	phantomStart := sub.Add(catmullPoints[0])

	v := catmullPoints[len(catmullPoints)-1].Sub(catmullPoints[len(catmullPoints)-2])
	phantomEnd := v.Add(catmullPoints[len(catmullPoints)-1])

	allPoints := make([]vector2d.Vector2dd, 0, len(catmullPoints)+2)
	allPoints = append(allPoints, phantomStart)
	allPoints = append(allPoints, catmullPoints...)
	allPoints = append(allPoints, phantomEnd)

	var bezierPoints []vector2d.Vector2dd
	for i := 1; i < len(allPoints)-2; i++ {
		p0 := allPoints[i-1]
		p1 := allPoints[i]
		p2 := allPoints[i+1]
		p3 := allPoints[i+2]

		// Catmull-Rom to Cubic Bezier conversion
		// Tangent magnitudes are scaled by 1/6 for uniform Catmull-Rom
		t1x := (p2.X - p0.X) / 6.0
		t1y := (p2.Y - p0.Y) / 6.0
		t2x := (p3.X - p1.X) / 6.0
		t2y := (p3.Y - p1.Y) / 6.0

		if i == 1 {
			bezierPoints = append(bezierPoints, p1)
		}
		bezierPoints = append(bezierPoints,
			vector2d.Vector2dd{X: p1.X + t1x, Y: p1.Y + t1y},
			vector2d.Vector2dd{X: p2.X - t2x, Y: p2.Y - t2y},
			p2,
		)
	}

	return bezierPoints
}
