package skills

import (
	"math"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

type Slider struct {
	Curve  []vector2d.Vector2dd
	Ncurve int
}

const CurvePointsSeparation = 5.0

// NewSlider replicates the original C++ Slider constructor, handling control point extraction
// and parsing individual curve segments.
func NewSlider(hitObject *osu.HitObject, line bool) *Slider {
	s := &Slider{}

	controlPointsCount := len(hitObject.Curves) + 1
	var points []vector2d.Vector2dd
	var lastPoi vector2d.Vector2dd = vector2d.New(-1.0, -1.0)

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

	var beziers []*Bezier

	// Splitting logic:
	// - If line is true (Linear slider): split into 2-point Beziers sequentially
	// - If line is false (Bezier slider): split into segments when encountering a duplicate (red) point
	for i := 0; i < controlPointsCount; i++ {
		tpoi := vector2d.New(getX(i), getY(i))
		if line {
			if !lastPoi.Equals(vector2d.New(-1.0, -1.0)) {
				points = append(points, tpoi)
				beziers = append(beziers, NewBezier(points))
				points = nil
			}
		} else if !lastPoi.Equals(vector2d.New(-1.0, -1.0)) && tpoi.Equals(lastPoi) {
			if len(points) >= 2 {
				beziers = append(beziers, NewBezier(points))
			}
			points = nil
		}
		points = append(points, tpoi)
		lastPoi = tpoi
	}

	if line || len(points) < 2 {
		// trying to continue Bezier with less than 2 points, ignore trailing point
	} else {
		beziers = append(beziers, NewBezier(points))
	}

	s.Init(beziers, hitObject)
	return s
}

// Init compiles multi-segment curves and generates exactly equidistant tracking points
func (s *Slider) Init(curvesList []*Bezier, hitObject *osu.HitObject) {
	s.Ncurve = int(hitObject.PixelLength / CurvePointsSeparation)
	s.Curve = make([]vector2d.Vector2dd, s.Ncurve+1)

	if len(curvesList) == 0 {
		curvesList = append(curvesList, NewBezier([]vector2d.Vector2dd{hitObject.Pos}))
		hitObject.EndPoint = hitObject.Pos
	}

	var distanceAt float64 = 0
	var curveCounter int = 0
	var curPoint int = 0

	curCurve := *curvesList[curveCounter]
	curveCounter++
	lastCurve := curCurve.curvePoints[0]
	var lastDistanceAt float64 = 0

	pixelLength := hitObject.PixelLength

	for i := 0; i < s.Ncurve+1; i++ {
		prefDistance := int(float64(i) * pixelLength / float64(s.Ncurve))
		for distanceAt < float64(prefDistance) {
			lastDistanceAt = distanceAt
			lastCurve = curCurve.curvePoints[curPoint]
			curPoint++

			if curPoint >= curCurve.curveCount {
				if curveCounter < len(curvesList) {
					curCurve = *curvesList[curveCounter]
					curveCounter++
					curPoint = 0
				} else {
					curPoint = curCurve.curveCount - 1
					if lastDistanceAt == distanceAt {
						break
					}
				}
			}
			distanceAt += curCurve.curveDis[curPoint]
		}
		thisCurve := curCurve.curvePoints[curPoint]

		// Interpolate intermediate positions
		if distanceAt-lastDistanceAt > 1.0 {
			t := (float64(prefDistance) - lastDistanceAt) / (distanceAt - lastDistanceAt)
			s.Curve[i] = vector2d.New(
				lerp(lastCurve.X, thisCurve.X, t),
				lerp(lastCurve.Y, thisCurve.Y, t),
			)
		} else {
			s.Curve[i] = thisCurve
		}
	}
}

// ApproximateSliderPoints calculates the dynamic repeat durations, end times,
// repeat checkpoints, ticks, and equidistant lerp points for all slider hit objects.
//
// Fits perfectly into your concurrency model by executing on the private *MapData instance context.
func approximateSliderPoints(md *MapData) {
	var timingPointOffsets []float64
	var beatLengths []float64
	var base float64

	for _, tp := range md.TimingPoints {
		timingPointOffsets = append(timingPointOffsets, float64(tp.Offset))
		if tp.Inherited {
			beatLengths = append(beatLengths, base)
		} else {
			beatLengths = append(beatLengths, tp.BeatInterval)
			base = tp.BeatInterval
		}
	}

	// Loop over each HitObject and process Slider-specific math
	for idx, _ := range md.Map.HitObjects {
		hitObject := &md.Map.HitObjects[idx]

		if hitObject.Type.IsHitSlider() {
			timingPointIndex := getValuePos(timingPointOffsets, float64(hitObject.Time))
			if timingPointIndex == -1 {
				timingPointIndex = 0
			}

			hitObject.ToRepeatTime = int(math.Round(-600/md.TimingPoints[timingPointIndex].Bpm*hitObject.PixelLength*md.TimingPoints[timingPointIndex].Sm) / (100.0 * md.Map.SliderMultiplier))
			hitObject.EndTime = hitObject.Time + hitObject.ToRepeatTime*hitObject.Repeat
			for i := hitObject.Time; i < hitObject.EndTime; i += hitObject.ToRepeatTime {
				if i > hitObject.EndTime {
					break
				}
				hitObject.RepeatTimes = append(hitObject.RepeatTimes, i)
			}

			tickInterval := int(beatLengths[timingPointIndex] / md.Map.SliderTickRate)
			const errInterval = 10
			j := 1

			for i := hitObject.Time + tickInterval; i < (hitObject.EndTime - errInterval); i += tickInterval {
				if i > hitObject.EndTime {
					break
				}

				tickTime := hitObject.Time + (tickInterval * j)
				if tickTime < 0 {
					break
				}

				hitObject.Ticks = append(hitObject.Ticks, tickTime)
				j++
			}

			if (absInt(hitObject.EndTime-hitObject.Time) < 100) && (len(hitObject.Ticks) == 0) {
				ho := &osu.HitObject{
					Curves: []vector2d.Vector2dd{
						vector2d.New(hitObject.Pos.X, hitObject.Pos.Y),
						vector2d.New(
							hitObject.Pos.X+float64(tickInterval)/md.Map.SliderTickRate,
							hitObject.Pos.Y+float64(tickInterval)/md.Map.SliderTickRate,
						),
					},
					Pos:          hitObject.Pos,
					Type:         hitObject.Type,
					Time:         hitObject.Time,
					EndTime:      hitObject.Time + 101,
					ToRepeatTime: hitObject.Time + 101,
					Repeat:       1,
					PixelLength:  100,
					CurveType:    osu.LinearCurve,
				}

				NewSlider(ho, true)
				md.Map.HitObjects[idx] = *ho
			}
		} else {
			hitObject.EndTime = hitObject.Time
		}
	}
}
