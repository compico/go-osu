package skills

import (
	"math"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

type CircumscribedCircle struct {
	Curve  []vector2d.Vector2dd
	Ncurve int
}

func NewCircumscribedCircle(hitObject *osu.HitObject) *CircumscribedCircle {
	cc := &CircumscribedCircle{}

	getX := func(i int) float64 {
		if i == 0 {
			return hitObject.Pos.X
		}
		return hitObject.Curves[i-1].X
	}

	getY := func(i int) float64 {
		if i == 0 {
			return hitObject.Pos.Y
		}
		return hitObject.Curves[i-1].Y
	}

	start := vector2d.New(getX(0), getY(0))
	mid := vector2d.New(getX(1), getY(1))
	end := vector2d.New(getX(2), getY(2))

	// Find circle center using perpendicular bisectors
	startMidPoint := start.MidPoint(mid)
	endMidPoint := end.MidPoint(mid)

	subStartMidPoint := mid.Sub(start)
	subEndMidPoint := mid.Sub(end)

	norStartMidPoint := subStartMidPoint.Nor()
	norEndMidPoint := subEndMidPoint.Nor()

	circleCenter := cc.intersect(startMidPoint, norStartMidPoint, endMidPoint, norEndMidPoint)
	if circleCenter.X == -1 && circleCenter.Y == -1 {
		// Fallback to linear slider
		slider := NewSlider(hitObject, true)
		cc.Curve = slider.Curve
		cc.Ncurve = slider.Ncurve
		return cc
	}

	startAngPoint := start.Sub(circleCenter)
	midAngPoint := mid.Sub(circleCenter)
	endAngPoint := end.Sub(circleCenter)

	startAng := math.Atan2(startAngPoint.Y, startAngPoint.X)
	midAng := math.Atan2(midAngPoint.Y, midAngPoint.X)
	endAng := math.Atan2(endAngPoint.Y, endAngPoint.X)

	twoPI := math.Pi * 2

	if !cc.isIn(startAng, midAng, endAng) {
		if math.Abs(startAng+twoPI-endAng) < twoPI && cc.isIn(startAng+twoPI, midAng, endAng) {
			startAng += twoPI
		} else if math.Abs(startAng-(endAng+twoPI)) < twoPI && cc.isIn(startAng, midAng, endAng+twoPI) {
			endAng += twoPI
		} else if math.Abs(startAng-twoPI-endAng) < twoPI && cc.isIn(startAng-twoPI, midAng, endAng) {
			startAng -= twoPI
		} else if math.Abs(startAng-(endAng-twoPI)) < twoPI && cc.isIn(startAng, midAng, endAng-twoPI) {
			endAng -= twoPI
		} else {
			return cc
		}
	}

	radius := startAngPoint.Length()
	pixelLength := hitObject.PixelLength
	arcAng := pixelLength / radius

	if endAng > startAng {
		endAng = startAng + arcAng
	} else {
		endAng = startAng - arcAng
	}

	step := hitObject.PixelLength / CurvePointsSeparation
	cc.Ncurve = int(step)
	length := cc.Ncurve + 1
	cc.Curve = make([]vector2d.Vector2dd, length)

	for i := 0; i < length; i++ {
		t := float64(i) / step
		ang := lerp(startAng, endAng, t)
		cc.Curve[i] = vector2d.New(
			math.Cos(ang)*radius+circleCenter.X,
			math.Sin(ang)*radius+circleCenter.Y,
		)
	}

	return cc
}

func (cc *CircumscribedCircle) isIn(a, b, c float64) bool {
	return (b > a && b < c) || (b < a && b > c)
}

func (cc *CircumscribedCircle) intersect(a, ta, b, tb vector2d.Vector2dd) vector2d.Vector2dd {
	des := tb.X*ta.Y - tb.Y*ta.X
	if math.Abs(des) < 0.00001 {
		return vector2d.New(-1.0, -1.0)
	}
	u := ((b.Y-a.Y)*ta.X + (a.X-b.X)*ta.Y) / des
	return vector2d.New(
		b.X+tb.X*u,
		b.Y+tb.Y*u,
	)
}
