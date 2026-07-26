package skills

import (
	"math"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

const OffsetMaxDisplacement = 2

// prepareMapData заполняет все необходимые поля MapData для расчёта скиллов.
// Должен вызываться ОДИН РАЗ перед любыми CalculateXxx().
func prepareMapData(md *MapData) {
	prepareTimingPoints(md)
	approximateSliderPoints(md)
	bakeSliderData(md)

	// PrepareAimData
	calculateMovementData(md)
	gatherTargetPoints(md)
	gatherAimAndReadingPoints(md)
	calculateAngles(md)

	// PrepareTapData
	calculatePressIntervals(md)
	gatherTapPatterns(md)
}

func calculatePressIntervals(md *MapData) {
	previousTime := -1
	for _, ho := range md.Map.HitObjects {
		if ho.Type.IsHitNormal() || ho.Type.IsHitSlider() {
			if previousTime != -1 {
				md.PressIntervals = append(md.PressIntervals, ho.Time-previousTime)
			}

			previousTime = ho.Time
		}
	}
}

func gatherAimAndReadingPoints(md *MapData) {
	index := 0
	prev := &md.Map.HitObjects[0]

	for i := range md.Map.HitObjects {
		ho := md.Map.HitObjects[i]

		if ho.Type.IsHitNormal() {
			md.AimPoints = append(md.AimPoints, AimPoint{
				Time: ho.Time,
				Pos:  ho.Pos,
				Type: AimPointCircle,
			})

			debugf("[AIMPOINT] idx=%v time=%v pos=(%v,%v) type=CIRCLE\n",
				index, ho.Time, ho.Pos.X, ho.Pos.Y)

			visTime := getVisibilityTimes(
				ho,
				md.Map.ApproachRate,
				md.HasAnyMods(),
				0.0,
				1.0,
			)

			md.ReadingPoints = append(md.ReadingPoints, ReadingPoint{
				Index:   index,
				Time:    ho.Time,
				Preempt: ho.Time - arToMs(md.Map.ApproachRate),
				FadeIn:  visTime.first,
				FadeOut: visTime.second,
				Angle:   0.0,
				Pos:     ho.Pos,
				Dist:    ho.Pos.DistanceFrom(prev.Pos),
			})

			index++
			prev = &md.Map.HitObjects[i]

		} else if ho.Type.IsHitSlider() {

			md.AimPoints = append(md.AimPoints, AimPoint{
				Time: ho.Time,
				Pos:  ho.Pos,
				Type: AimPointSlider,
			})

			debugf("[AIMPOINT] idx=%v time=%v pos=(%v,%v) type=SLIDER_START ticks=%v repeat=%v\n",
				index, ho.Time, ho.Pos.X, ho.Pos.Y, len(ho.Ticks), ho.Repeat)

			visTime := getVisibilityTimes(
				ho,
				md.Map.ApproachRate,
				md.HasAnyMods(),
				0.0,
				1.0,
			)

			md.ReadingPoints = append(md.ReadingPoints, ReadingPoint{
				Index:   index,
				Time:    ho.Time,
				Preempt: ho.Time - arToMs(md.Map.ApproachRate),
				FadeIn:  visTime.first,
				FadeOut: visTime.second,
				Angle:   0.0,
				Pos:     ho.Pos,
				Dist:    ho.Pos.DistanceFrom(prev.Pos),
			})

			index++
			prev = &md.Map.HitObjects[i]

			endTime := GetLastTickTime(ho)
			endPos := getSliderPos(ho, endTime)

			dist := ho.Pos.DistanceFrom(endPos)
			threshold := 2 * float64(cs2px(md.Map.CircleSize))

			debugf("[AIMPOINT] slider check: endTime=%v endPos=(%v,%v) ticks=%v distToStart=%v threshold=%v willAdd=%v\n",
				endTime, endPos.X, endPos.Y, len(ho.Ticks), dist, threshold, (len(ho.Ticks) > 0 || dist > threshold))

			if len(ho.Ticks) > 0 || dist > threshold {

				md.AimPoints = append(md.AimPoints, AimPoint{
					Time: endTime,
					Pos:  endPos,
					Type: AimPointSliderEnd,
				})

				debugf("[AIMPOINT] idx=%v time=%v pos=(%v,%v) type=SLIDEREND\n",
					index, endTime, endPos.X, endPos.Y)

				md.ReadingPoints = append(md.ReadingPoints, ReadingPoint{
					Index:   index,
					Time:    endTime,
					Preempt: endTime - arToMs(md.Map.ApproachRate),
					FadeIn:  visTime.first,
					FadeOut: visTime.second,
					Angle:   0.0,
					Pos:     endPos,
					Dist:    endPos.DistanceFrom(prev.Pos),
				})

				index++
				prev = &md.Map.HitObjects[i]
			}
		}
	}
	debugf("[AIMPOINT] total aimPoints=%v readingPoints=%v\n", len(md.AimPoints), len(md.ReadingPoints))
}

func getSliderPos(ho osu.HitObject, time int) vector2d.Vector2dd {
	if ho.Type.IsHitSlider() {
		percent := 0.0

		if time <= ho.Time {
			percent = 0.0
		} else if time > ho.EndTime {
			percent = 1.0
		} else {
			timeLength := time - ho.Time
			repeatsDone := timeLength / ho.ToRepeatTime
			percent = float64(timeLength-ho.ToRepeatTime*repeatsDone) / float64(ho.ToRepeatTime)
			if (repeatsDone % 2) == 1 {
				percent = 1 - percent
			}
		}

		indexF := percent * float64(ho.Ncurve)
		index := int(indexF)

		if index >= ho.Ncurve {
			return ho.LerpPoints[ho.Ncurve]
		}

		t2 := indexF - float64(index)

		return vector2d.Vector2dd{
			X: lerp(ho.LerpPoints[index].X, ho.LerpPoints[index+1].X, t2),
			Y: lerp(ho.LerpPoints[index].Y, ho.LerpPoints[index+1].Y, t2),
		}
	}

	return vector2d.Vector2dd{
		X: -1, Y: -1,
	}
}

func GetLastTickTime(ho osu.HitObject) int {
	if !(len(ho.Ticks) > 0) {
		if ho.Repeat > 1 {
			return int(float64(ho.EndTime) - float64(ho.EndTime-ho.RepeatTimes[len(ho.RepeatTimes)-1])/2.0)
		}
		return int(float64(ho.EndTime) - float64(ho.EndTime-ho.Time)/2.0)
	}

	return int(float64(ho.EndTime) - float64(ho.EndTime-ho.Ticks[len(ho.Ticks)-1])/2.0)
}

func prepareTimingPoints(md *MapData) {
	src := md.Map.TimingPoints
	md.TimingPoints = make([]TimingPoint, 0, len(src))

	md.BpmMin = 10000
	md.BpmMax = 0

	state := timingState{
		bpm:   0,
		sm:    -100,
		oldSM: -100,
	}

	for _, original := range src {
		tp := createTimingPoint(original)

		state.update(&tp)

		tp.Bpm = state.bpm
		tp.Sm = state.sm

		if !tp.Inherited {
			if tp.Bpm < md.BpmMin {
				md.BpmMin = tp.Bpm
			}
			if tp.Bpm > md.BpmMax {
				md.BpmMax = tp.Bpm
			}
		}

		md.TimingPoints = append(md.TimingPoints, tp)
	}
}

func createTimingPoint(src osu.TimingPoint) TimingPoint {
	return TimingPoint{
		Offset:       int(src.Time),
		BeatInterval: src.BeatLength,
		Meter:        src.Meter,
		Inherited:    !src.Uninherited,
	}
}

type timingState struct {
	bpm   float64
	sm    float64
	oldSM float64
}

func (s *timingState) update(tp *TimingPoint) {
	if tp.Inherited {
		s.updateSliderVelocity(tp)

		return
	}

	s.updateBPM(tp)
}

func (s *timingState) updateSliderVelocity(tp *TimingPoint) {
	if tp.BeatInterval <= 0 {
		s.sm = tp.BeatInterval
		s.oldSM = tp.BeatInterval
		return
	}

	s.sm = s.oldSM
}

func (s *timingState) updateBPM(tp *TimingPoint) {
	s.sm = -100

	if tp.BeatInterval <= 0 {
		return
	}

	s.bpm = 60000 / tp.BeatInterval
}

// bakeSliderData parses and pre-calculates appropriate geometric curve points for all sliders
func bakeSliderData(md *MapData) {
	if md.Map == nil {
		return
	}

	for k, h := range md.Map.HitObjects {
		if h.Type.IsHitSlider() {
			switch h.CurveType {
			case osu.BezierCurve:
				slider := NewSlider(&h, false)
				md.Map.HitObjects[k].LerpPoints = slider.Curve
				md.Map.HitObjects[k].Ncurve = slider.Ncurve

				break
			case osu.PerfectCurve:
				if len(h.Curves) == 2 {
					circle := NewCircumscribedCircle(&h)
					md.Map.HitObjects[k].LerpPoints = circle.Curve
					md.Map.HitObjects[k].Ncurve = circle.Ncurve
				} else {
					slider := NewSlider(&h, false)
					md.Map.HitObjects[k].LerpPoints = slider.Curve
					md.Map.HitObjects[k].Ncurve = slider.Ncurve
				}

				break
			case osu.LinearCurve, osu.CatmullCurve:
				slider := NewSlider(&h, true)
				md.Map.HitObjects[k].LerpPoints = slider.Curve
				md.Map.HitObjects[k].Ncurve = slider.Ncurve

				break
			}

			// Assign proper endpoint based on whether repeat flips the slider or not
			lerpPoints := md.Map.HitObjects[k].LerpPoints
			if len(lerpPoints) > 0 {
				if h.Repeat%2 != 0 {
					md.Map.HitObjects[k].EndPoint = lerpPoints[len(lerpPoints)-1]
				} else {
					md.Map.HitObjects[k].EndPoint = h.Pos
				}
			} else {
				md.Map.HitObjects[k].EndPoint = h.Pos
			}
		}
	}
}

type MovementData struct {
	Distances  []float64
	Velocities Velocities
}

// calculateMovementData computes speed, distance, vector velocity, and acceleration changes
func calculateMovementData(md *MapData) {
	if md.Map == nil || len(md.Map.HitObjects) == 0 {
		return
	}

	var previousPos vector2d.Vector2dd
	previousTime := -1

	for _, ho := range md.Map.HitObjects {
		if ho.Type.IsHitNormal() || ho.Type.IsHitSlider() {
			if previousTime != -1 {
				dx := ho.Pos.X - previousPos.X
				dy := ho.Pos.Y - previousPos.Y
				dist := math.Sqrt(dx*dx + dy*dy)

				// Subtract radii of overlap
				radSubtract := 2.0 * float64(cs2px(md.Map.CircleSize))
				if dist >= radSubtract {
					dist -= radSubtract
				} else {
					dist /= 2.0
				}

				interval := float64(ho.Time - previousTime)
				if interval <= 0 {
					interval = 1.0 // Prevent NaN
				}

				md.Distances = append(md.Distances, dist)
				md.Velocities.X = append(md.Velocities.X, dx/interval)
				md.Velocities.Y = append(md.Velocities.Y, dy/interval)
			}

			previousPos = ho.Pos
			previousTime = ho.Time
		}
	}

	var OldVelX float64 = 0
	var OldVelY float64 = 0

	for i := 0; i < len(md.Velocities.X); i++ {
		vX := md.Velocities.X[i]
		vY := md.Velocities.Y[i]
		if i > 0 {
			md.Velocities.Xchange = append(md.Velocities.Xchange, vX-OldVelX)
			md.Velocities.Ychange = append(md.Velocities.Ychange, vY-OldVelY)
		}
		OldVelX = vX
		OldVelY = vY
	}
}

func gatherTapPatterns(md *MapData) {
	old := 0
	var tmp []int
	i := 0
	uniq := make(map[int]struct{})
	for _, interval := range md.PressIntervals {
		_, it := uniq[interval]

		if !it {
			found := false
			for p := interval - OffsetMaxDisplacement; p < (interval + OffsetMaxDisplacement); p++ {
				_, it2 := uniq[p]
				if it2 {
					interval = p
					found = true
					break
				}
			}
			if !found {
				uniq[interval] = struct{}{}
				md.Streams[interval] = make([][]int, 0)
				md.Bursts[interval] = make([][]int, 0)
			}
		}

		if math.Abs(float64(interval-old)) > OffsetMaxDisplacement {
			if len(tmp) > 1 {
				if len(tmp) > 6 {
					md.Streams[old] = append(md.Streams[old], tmp)
				} else {
					md.Bursts[old] = append(md.Bursts[old], tmp)
				}
			}
			tmp = []int{}
		}

		tmp = append(tmp, md.Map.HitObjects[i].Time)
		old = interval
		i++
	}

	if len(tmp) > 1 {
		if len(tmp) > 6 {
			md.Streams[old] = append(md.Streams[old], tmp)
		} else {
			md.Bursts[old] = append(md.Bursts[old], tmp)
		}
	}
}

// gatherTargetPoints ports generic.cpp's GatherTargetPoints.
//
// Deviation note (bug-compatible on purpose): Key is set from `i`, a
// counter that only increments for hit objects that pass the <5ms
// proximity filter below — it is NOT the object's real index into
// Map.HitObjects whenever a filter skip has happened earlier in the map.
// The original C++ has this exact same property (its `i` only increments
// past the `continue`), and reaction.cpp later does
// `hitobjects[targetpoint.key].time` assuming they line up. This is
// almost never observable in practice (two hit objects within 5ms of each
// other is rare), but it's preserved here for numeric parity with the
// reference implementation rather than silently "fixed".
func gatherTargetPoints(md *MapData) {
	i := 0
	prevTime := math.MinInt32

	for idx := range md.Map.HitObjects {
		ho := &md.Map.HitObjects[idx]

		// Hack around objects that need to be hit impossibly fast.
		if absInt(ho.Time-prevTime) < 5 {
			continue // filter out hitobjects that occur less than 5ms after another
		}
		prevTime = ho.Time

		if ho.Type.IsHitNormal() {
			md.TargetPoints = append(md.TargetPoints, TargetPoint{
				Time: float64(ho.Time), Pos: ho.Pos, Key: i, Press: false,
			})
		} else if ho.Type.IsHitSlider() {
			for _, tick := range ho.Ticks {
				pos := getSliderPos(*ho, tick)
				md.TargetPoints = append(md.TargetPoints, TargetPoint{
					Time: float64(tick), Pos: pos, Key: i, Press: true,
				})
			}
		}

		i++
	}
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// roundInterval rounds an interval to the nearest bucket (in ms).
func roundInterval(interval, bucket int) int {
	return ((interval + bucket/2) / bucket) * bucket
}

// findTimingAt ports reaction.cpp's FindTimingAt: binary search for the
// TargetPoints interval containing time.
//
// Deviation: the C++ falls through, on an unmatched search, to returning
// (int)NAN — casting NaN to int is undefined behavior in C++ and not
// reproducible in Go. That branch is unreachable given the binary search
// invariants for any well-formed input (every time either falls within
// some interval, or triggers one of the two out-of-range checks above
// it), so we return 0 there instead of attempting to replicate UB.
func findTimingAt(timings []TargetPoint, time float64) int {
	start := 0
	end := len(timings) - 2
	if end < 0 {
		return 0
	}

	for start <= end {
		mid := (start + end) / 2
		if btwn(timings[mid].Time, time, timings[mid+1].Time) {
			return mid + 1
		}
		if time < timings[mid].Time {
			end = mid - 1
		} else {
			start = mid + 1
		}
	}

	if time < timings[0].Time {
		return math.MinInt32
	}
	if time > timings[len(timings)-1].Time {
		return math.MaxInt32
	}
	return 0
}

func calculateAngles(md *MapData) {
	for i := 0; i+2 < len(md.AimPoints); i++ {
		angle := getDirAngle(
			md.AimPoints[i].Pos,
			md.AimPoints[i+1].Pos,
			md.AimPoints[i+2].Pos,
		)

		md.Angles = append(md.Angles, angle)
		md.ReadingPoints[i+2].Angle = angle
	}

	if !(len(md.Angles) > 0) {
		return
	}

	oldAngle := md.Angles[0] - 2*md.Angles[0]
	for _, angle := range md.Angles {
		bonus := calculateAngleBonus(angle, oldAngle)

		md.AngleBonuses = append(md.AngleBonuses, bonus)
		oldAngle = angle
	}
}

func calculateAngleBonus(angle, oldAngle float64) float64 {
	absAngle := math.Abs(angle)

	if sign(angle) == sign(oldAngle) {
		if absAngle < 90 {
			return math.Sin(degToRad(absAngle)*0.784 + 0.339837)
		}

		return math.Sin(degToRad(absAngle))
	}

	if absAngle < 90 {
		return math.Sin(degToRad(absAngle)*0.536 + 0.72972)
	}

	return math.Sin(degToRad(absAngle)) / 2
}

func getVisibilityTimes(
	ho osu.HitObject,
	ar float64,
	hidden bool,
	opacityStart float64,
	opacityEnd float64,
) struct{ first, second int } {
	preemptTime := float64(ho.Time - arToMs(ar))

	var times struct {
		first  int
		second int
	}

	if hidden {
		fadeInDuration := 0.4 * float64(arToMs(ar))
		fadeInTimeEnd := preemptTime + fadeInDuration

		times.first = int(getValue(preemptTime, fadeInTimeEnd, opacityStart))

		if ho.Type.IsHitSlider() {
			fadeOutDuration := 0.7 * (float64(ho.Time) - fadeInTimeEnd)
			fadeOutTimeEnd := fadeInTimeEnd + fadeOutDuration

			times.second = int(
				getValue(
					fadeInTimeEnd,
					fadeOutTimeEnd,
					1.0-opacityStart,
				),
			)

			return times
		}

		fadeOutDuration := float64(ho.EndTime) - fadeInTimeEnd
		fadeOutTimeEnd := fadeInTimeEnd + fadeOutDuration

		times.second = int(
			getValue(
				fadeInTimeEnd,
				fadeOutTimeEnd,
				1.0-opacityEnd,
			),
		)

		return times
	}

	fadeInDuration := float64(min(arToMs(ar), 400))
	fadeInTimeEnd := preemptTime + fadeInDuration

	times.first = int(
		getValue(
			preemptTime,
			fadeInTimeEnd,
			opacityStart,
		),
	)

	if ho.Type.IsHitSlider() {
		times.second = ho.EndTime

		return times
	}

	times.second = ho.Time

	return times
}
