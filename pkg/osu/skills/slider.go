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
	if md.Map == nil {
		return
	}

	var timingPointOffsets []float64
	var beatLengths []float64
	var base float64

	// Read from skills.TimingPoints if populated, otherwise fallback to the raw osu.Beatmap timing points

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

		// HitSlider flags = 2 (standard osu slider flag bit)
		isSlider := (int(hitObject.Type) & 2) != 0

		if isSlider {
			timingPointIndex := getValuePos(timingPointOffsets, float64(hitObject.Time))
			if timingPointIndex == -1 {
				timingPointIndex = 0
			}

			var bpm float64
			var sm float64

			// Retrieve proper BPM and Slider Velocity Multiplier for this segment
			if len(md.TimingPoints) > 0 && timingPointIndex < len(md.TimingPoints) {
				tp := md.TimingPoints[timingPointIndex]
				bpm = tp.Bpm
				sm = tp.Sm
			} else if len(md.Map.TimingPoints) > 0 && timingPointIndex < len(md.Map.TimingPoints) {
				tp := &md.Map.TimingPoints[timingPointIndex]
				bpm = tp.BPM()
				sm = tp.SliderVelocity()
			}

			sliderMultiplier := md.Map.SliderMultiplier
			if sliderMultiplier <= 0 {
				sliderMultiplier = 1.0
			}

			if bpm <= 0 {
				bpm = 120
			}

			// Normalized, safe positive-only repeat calculation equivalent to the original C++:
			// ((-600.0 / bpm) * hitObject.pixelLength * sm) / (100.0 * beatmap.sm)
			val := ((-600.0 / bpm) *
				hitObject.PixelLength *
				sm) /
				(100.0 * md.Map.SliderMultiplier)

			hitObject.ToRepeatTime = int(math.Round(val))

			hitObject.EndTime = hitObject.Time + hitObject.ToRepeatTime*hitObject.Repeat

			// Calculate precise timestamps of each slider repeat turnaround
			hitObject.RepeatTimes = nil
			if hitObject.Repeat > 1 {
				for t := hitObject.Time; t < hitObject.EndTime; t += hitObject.ToRepeatTime {
					if t > hitObject.EndTime {
						break
					}
					hitObject.RepeatTimes = append(hitObject.RepeatTimes, t)
				}
			}

			// Calculate Slider Ticks
			var beatLength float64
			if timingPointIndex < len(beatLengths) {
				beatLength = beatLengths[timingPointIndex]
			} else {
				beatLength = 60000.0 / bpm
			}

			sliderTickRate := md.Map.SliderTickRate
			if sliderTickRate <= 0 {
				sliderTickRate = 1.0
			}

			tickInterval := int(beatLength / sliderTickRate)
			errInterval := 10
			j := 1

			hitObject.Ticks = nil
			for t := hitObject.Time + tickInterval; t < (hitObject.EndTime - errInterval); t += tickInterval {
				if t > hitObject.EndTime {
					break
				}
				tickTime := hitObject.Time + int(float64(tickInterval)*float64(j))
				if tickTime < 0 {
					break
				}
				hitObject.Ticks = append(hitObject.Ticks, tickTime)
				j++
			}

			// Short slider check: If the slider starts and ends in < 100ms and has no ticks to allow a sliderbreak,
			// convert it to a brief, standard straight linear slider to protect player combo scoring
			if (math.Abs(float64(hitObject.EndTime-hitObject.Time)) < 100) && len(hitObject.Ticks) == 0 {
				hitObjectNew := osu.HitObject{
					Pos: hitObject.Pos,
					Curves: []vector2d.Vector2dd{
						hitObject.Pos,
						vector2d.New(
							hitObject.Pos.X+float64(tickInterval)/sliderTickRate,
							hitObject.Pos.Y+float64(tickInterval)/sliderTickRate,
						),
					},
					Type:         hitObject.Type,
					Time:         hitObject.Time,
					EndTime:      hitObject.Time + 101,
					ToRepeatTime: hitObject.Time + 101,
					Repeat:       1,
					PixelLength:  100,
					CurveType:    'L', // Force linear
				}

				slider := NewSlider(&hitObjectNew, true)
				hitObjectNew.LerpPoints = slider.Curve
				hitObjectNew.Ncurve = slider.Ncurve
				if len(slider.Curve) > 0 {
					hitObjectNew.EndPoint = slider.Curve[len(slider.Curve)-1]
				} else {
					hitObjectNew.EndPoint = hitObjectNew.Pos
				}

				*hitObject = hitObjectNew
				continue
			}

			// Standard Bezier/Linear approximation
			slider := NewSlider(hitObject, hitObject.CurveType == 'L')
			hitObject.LerpPoints = slider.Curve
			hitObject.Ncurve = slider.Ncurve
			if len(slider.Curve) > 0 {
				hitObject.EndPoint = slider.Curve[len(slider.Curve)-1]
			} else {
				hitObject.EndPoint = hitObject.Pos
			}

		} else {
			// Circle or Spinner
			hitObject.EndTime = hitObject.Time
		}
	}
}
