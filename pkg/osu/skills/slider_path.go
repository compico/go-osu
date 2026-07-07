package skills

import (
	"math"
	"sort"

	"github.com/compico/go-osu/pkg/vector2d"
)

// PathType mirrors the curve types used in osu! beatmaps.
type PathType int

const (
	PathTypeLinear PathType = iota
	PathTypeBezier
	PathTypePerfect
	PathTypeCatmull
)

// PathSegment represents a single continuous curve segment within a slider.
type PathSegment struct {
	Type   PathType
	Points []vector2d.Vector2dd
}

// SliderPath calculates and stores the approximated path of a slider.
// It allows querying the exact position at any given distance along the slider.
type SliderPath struct {
	Segments          []PathSegment
	CalculatedPath    []vector2d.Vector2dd
	CumulativeLengths []float64
	ExpectedLength    float64
}

// NewSliderPath constructs a SliderPath from the given control points and curve type.
func NewSliderPath(points []vector2d.Vector2dd, curveType rune, expectedLength float64) *SliderPath {
	sp := &SliderPath{
		ExpectedLength: expectedLength,
	}

	pt := PathTypeBezier
	switch curveType {
	case 'L':
		pt = PathTypeLinear
	case 'P':
		pt = PathTypePerfect
	case 'C':
		pt = PathTypeCatmull
	case 'B':
		pt = PathTypeBezier
	default:
		pt = PathTypeBezier
	}

	// In osu! v14+, curve types can be specified per red anchor.
	// For simplicity and compatibility with our current HitObject struct,
	// we assume the entire slider follows the primary CurveType unless
	// future parser updates provide per-point types.
	sp.Segments = append(sp.Segments, PathSegment{
		Type:   pt,
		Points: points,
	})

	sp.calculatePath()
	sp.calculateCumulativeLengths()

	return sp
}

func (sp *SliderPath) calculatePath() {
	sp.CalculatedPath = nil
	for _, seg := range sp.Segments {
		var subPath []vector2d.Vector2dd
		switch seg.Type {
		case PathTypeLinear:
			subPath = seg.Points
		case PathTypeBezier:
			subPath = approximateBezier(seg.Points, 0.25)
		case PathTypePerfect:
			subPath = approximateCircle(seg.Points, 0.25)
			if len(subPath) == 0 {
				// Fallback to Bezier if points are collinear or invalid for a perfect circle
				subPath = approximateBezier(seg.Points, 0.25)
			}
		case PathTypeCatmull:
			subPath = approximateCatmull(seg.Points, 0.25)
		}

		if len(sp.CalculatedPath) == 0 {
			sp.CalculatedPath = subPath
		} else {
			// Avoid duplicate points at segment boundaries
			if len(subPath) > 0 && len(sp.CalculatedPath) > 0 {
				if sp.CalculatedPath[len(sp.CalculatedPath)-1].Equals(subPath[0]) {
					sp.CalculatedPath = append(sp.CalculatedPath, subPath[1:]...)
				} else {
					sp.CalculatedPath = append(sp.CalculatedPath, subPath...)
				}
			}
		}
	}
}

func (sp *SliderPath) calculateCumulativeLengths() {
	sp.CumulativeLengths = make([]float64, len(sp.CalculatedPath))
	if len(sp.CalculatedPath) == 0 {
		return
	}
	sp.CumulativeLengths[0] = 0
	for i := 1; i < len(sp.CalculatedPath); i++ {
		dist := sp.CalculatedPath[i].DistanceFrom(sp.CalculatedPath[i-1])
		sp.CumulativeLengths[i] = sp.CumulativeLengths[i-1] + dist
	}
}

// PositionAt returns the exact coordinates at a given distance along the slider path.
func (sp *SliderPath) PositionAt(distance float64) vector2d.Vector2dd {
	if len(sp.CalculatedPath) == 0 {
		return vector2d.Vector2dd{}
	}
	if distance <= 0 {
		return sp.CalculatedPath[0]
	}
	totalLen := sp.CumulativeLengths[len(sp.CumulativeLengths)-1]
	if distance >= totalLen {
		return sp.CalculatedPath[len(sp.CalculatedPath)-1]
	}

	// Binary search for the correct segment
	i := sort.SearchFloat64s(sp.CumulativeLengths, distance)
	if i == 0 {
		return sp.CalculatedPath[0]
	}

	// Linear interpolation between the two surrounding calculated points
	start := sp.CalculatedPath[i-1]
	end := sp.CalculatedPath[i]
	startLen := sp.CumulativeLengths[i-1]
	endLen := sp.CumulativeLengths[i]

	t := (distance - startLen) / (endLen - startLen)
	return vector2d.Vector2dd{
		X: start.X + t*(end.X-start.X),
		Y: start.Y + t*(end.Y-start.Y),
	}
}

// TotalLength returns the total calculated length of the path.
func (sp *SliderPath) TotalLength() float64 {
	if len(sp.CumulativeLengths) == 0 {
		return 0
	}
	return sp.CumulativeLengths[len(sp.CumulativeLengths)-1]
}

// --- Curve Approximation Algorithms ---

// deCasteljau performs one step of De Casteljau's algorithm, splitting a Bezier
// curve at parameter t into two sub-curves (left and right).
func deCasteljau(points []vector2d.Vector2dd, t float64) ([]vector2d.Vector2dd, []vector2d.Vector2dd) {
	n := len(points)
	if n == 0 {
		return nil, nil
	}

	left := make([]vector2d.Vector2dd, n)
	right := make([]vector2d.Vector2dd, n)

	current := make([]vector2d.Vector2dd, n)
	copy(current, points)

	left[0] = current[0]
	right[n-1] = current[n-1]

	for level := 1; level < n; level++ {
		for i := 0; i < n-level; i++ {
			current[i].X = current[i].X*(1-t) + current[i+1].X*t
			current[i].Y = current[i].Y*(1-t) + current[i+1].Y*t
		}
		left[level] = current[0]
		right[n-1-level] = current[n-1-level]
	}

	return left, right
}

// approximateBezier recursively subdivides a Bezier curve until it is flat enough.
func approximateBezier(points []vector2d.Vector2dd, tolerance float64) []vector2d.Vector2dd {
	var result []vector2d.Vector2dd

	var subdivide func([]vector2d.Vector2dd)
	subdivide = func(pts []vector2d.Vector2dd) {
		isFlat := true
		if len(pts) > 2 {
			start := pts[0]
			end := pts[len(pts)-1]
			dx := end.X - start.X
			dy := end.Y - start.Y
			length := math.Sqrt(dx*dx + dy*dy)

			if length > 0 {
				dx /= length
				dy /= length
				for i := 1; i < len(pts)-1; i++ {
					vx := pts[i].X - start.X
					vy := pts[i].Y - start.Y
					// Distance from point to the line (start, end)
					dist := math.Abs(vx*dy - vy*dx)
					if dist > tolerance {
						isFlat = false
						break
					}
				}
			} else {
				for i := 1; i < len(pts)-1; i++ {
					if pts[i].DistanceFrom(start) > tolerance {
						isFlat = false
						break
					}
				}
			}
		}

		if isFlat {
			if len(result) == 0 || !result[len(result)-1].Equals(pts[0]) {
				result = append(result, pts[0])
			}
			result = append(result, pts[len(pts)-1])
			return
		}

		left, right := deCasteljau(pts, 0.5)
		subdivide(left)
		subdivide(right)
	}

	subdivide(points)
	return result
}

// approximateCircle approximates a Perfect Curve (circumscribed circle) through 3 points.
// Returns nil if the points are collinear or invalid.
func approximateCircle(points []vector2d.Vector2dd, tolerance float64) []vector2d.Vector2dd {
	if len(points) != 3 {
		return nil // osu! Perfect curves are strictly defined by 3 points
	}

	a := points[0]
	b := points[1]
	c := points[2]

	d := 2 * (a.X*(b.Y-c.Y) + b.X*(c.Y-a.Y) + c.X*(a.Y-b.Y))
	if math.Abs(d) < 1e-6 {
		return nil // Collinear points
	}

	ux := ((a.X*a.X+a.Y*a.Y)*(b.Y-c.Y) + (b.X*b.X+b.Y*b.Y)*(c.Y-a.Y) + (c.X*c.X+c.Y*c.Y)*(a.Y-b.Y)) / d
	uy := ((a.X*a.X+a.Y*a.Y)*(c.X-b.X) + (b.X*b.X+b.Y*b.Y)*(a.X-c.X) + (c.X*c.X+c.Y*c.Y)*(b.X-a.X)) / d

	center := vector2d.Vector2dd{X: ux, Y: uy}
	radius := center.DistanceFrom(a)

	startAngle := math.Atan2(a.Y-center.Y, a.X-center.X)
	midAngle := math.Atan2(b.Y-center.Y, b.X-center.X)
	endAngle := math.Atan2(c.Y-center.Y, c.X-center.X)

	normalize := func(angle float64) float64 {
		for angle < 0 {
			angle += 2 * math.Pi
		}
		for angle >= 2*math.Pi {
			angle -= 2 * math.Pi
		}
		return angle
	}

	start := normalize(startAngle)
	mid := normalize(midAngle)
	end := normalize(endAngle)

	isPositive := false
	if start < end {
		if mid > start && mid < end {
			isPositive = true
		}
	} else {
		if mid > start || mid < end {
			isPositive = true
		}
	}

	totalAngle := end - start
	if isPositive {
		if totalAngle < 0 {
			totalAngle += 2 * math.Pi
		}
	} else {
		if totalAngle > 0 {
			totalAngle -= 2 * math.Pi
		}
	}

	// Number of points based on angle (approx 1 point per 0.1 radian)
	numPoints := int(math.Ceil(math.Abs(totalAngle) / 0.1))
	if numPoints < 2 {
		numPoints = 2
	}

	var result []vector2d.Vector2dd
	for i := 0; i <= numPoints; i++ {
		t := float64(i) / float64(numPoints)
		angle := start + t*totalAngle
		result = append(result, vector2d.Vector2dd{
			X: center.X + radius*math.Cos(angle),
			Y: center.Y + radius*math.Sin(angle),
		})
	}

	return result
}

// approximateCatmull approximates a Catmull-Rom spline.
func approximateCatmull(points []vector2d.Vector2dd, tolerance float64) []vector2d.Vector2dd {
	if len(points) < 2 {
		return points
	}

	var result []vector2d.Vector2dd
	result = append(result, points[0])

	for i := 0; i < len(points)-1; i++ {
		p1 := points[i]
		p2 := points[i+1]

		var p0, p3 vector2d.Vector2dd
		if i > 0 {
			p0 = points[i-1]
		} else {
			p0 = p1
		}

		if i < len(points)-2 {
			p3 = points[i+2]
		} else {
			p3 = p2
		}

		numPoints := 20 // Fixed resolution for Catmull segments
		for j := 1; j <= numPoints; j++ {
			t := float64(j) / float64(numPoints)
			t2 := t * t
			t3 := t2 * t

			x := 0.5 * ((2 * p1.X) +
				(-p0.X+p2.X)*t +
				(2*p0.X-5*p1.X+4*p2.X-p3.X)*t2 +
				(-p0.X+3*p1.X-3*p2.X+p3.X)*t3)

			y := 0.5 * ((2 * p1.Y) +
				(-p0.Y+p2.Y)*t +
				(2*p0.Y-5*p1.Y+4*p2.Y-p3.Y)*t2 +
				(-p0.Y+3*p1.Y-3*p2.Y+p3.Y)*t3)

			result = append(result, vector2d.Vector2dd{X: x, Y: y})
		}
	}

	return result
}
