package skills

import (
	"math"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

// PrepareMapData заполняет все необходимые поля MapDataы для расчёта скиллов.
// Должен вызываться ОДИН РАЗ перед любыми CalculateXxx().
func PrepareMapData(md *MapData) {
	gatherAimPoints(md)
	calculateAnglesAndBonuses(md)
	populateReadingPoints(md)
	gatherTapIntervals(md)
}

// calculateSliderDuration вычисляет длительность слайдера в мс на основе BPM и SliderMultiplier.
func calculateSliderDuration(ho *osu.HitObject, bm *osu.Beatmap) int {
	var tp *osu.TimingPoint
	for i := len(bm.TimingPoints) - 1; i >= 0; i-- {
		if bm.TimingPoints[i].Time <= float64(ho.Time) {
			tp = &bm.TimingPoints[i]
			break
		}
	}
	if tp == nil {
		return 0
	}

	var baseTp *osu.TimingPoint
	for i := len(bm.TimingPoints) - 1; i >= 0; i-- {
		if bm.TimingPoints[i].Uninherited && bm.TimingPoints[i].Time <= float64(ho.Time) {
			baseTp = &bm.TimingPoints[i]
			break
		}
	}
	if baseTp == nil || baseTp.BeatLength <= 0 {
		return 0
	}

	velocity := (bm.SliderMultiplier * 100.0 / baseTp.BeatLength) * tp.SliderVelocity()
	if velocity <= 0 {
		return 0
	}

	return int(ho.PixelLength / velocity)
}

// gatherAimPoints converts HitObjects into a flat list of AimPoints,
// expanding sliders into their constituent parts (start, reverses, end)
// using the new SliderPath geometry logic.
func gatherAimPoints(md *MapData) {
	var aimPoints []AimPoint
	timeMapper := make(map[int]int)

	for i := range md.Map.HitObjects {
		ho := &md.Map.HitObjects[i]
		if ho.Type&int(osu.HitSpinner) != 0 {
			md.Spinners++
			continue
		}

		// The original C++ code records a single aim point for the hit object
		// itself, and for sliders adds an end point only when it meaningfully
		// differs from the start (e.g. a real slider tail).
		pointType := AimPointCircle
		if ho.Type&int(osu.HitSlider) != 0 {
			pointType = AimPointSlider
		}

		aimPoints = append(aimPoints, AimPoint{
			Time: ho.Time,
			Pos:  ho.Pos,
			Type: pointType,
		})
		timeMapper[ho.Time] = len(aimPoints) - 1

		if ho.Type&int(osu.HitSlider) != 0 {
			points := append([]vector2d.Vector2dd{ho.Pos}, ho.Curves...)
			sp := NewSliderPath(points, ho.CurveType, ho.PixelLength)

			duration := calculateSliderDuration(ho, md.Map)
			ho.EndTime = ho.Time + duration
			ho.RepeatTimes = calculateRepeatTimes(ho, duration)

			endPos := sp.PositionAt(ho.PixelLength)
			if len(ho.Ticks) > 0 || ho.Pos.DistanceFrom(endPos) > 2*cs2px(md.Map.CircleSize) {
				aimPoints = append(aimPoints, AimPoint{
					Time: ho.EndTime,
					Pos:  endPos,
					Type: AimPointSliderEnd,
				})
				timeMapper[ho.EndTime] = len(aimPoints) - 1
			}

			ho.EndPoint = endPos
		}
	}

	md.AimPoints = aimPoints
	md.TimeMapper = timeMapper
}

// calculateRepeatTimes determines the timing for slider repeats.
func calculateRepeatTimes(ho *osu.HitObject, duration int) []int {
	if ho.Repeat <= 1 || duration <= 0 {
		return nil
	}
	segmentDuration := duration / ho.Repeat
	times := make([]int, ho.Repeat-1)
	for i := 0; i < ho.Repeat-1; i++ {
		times[i] = ho.Time + (i+1)*segmentDuration
	}
	return times
}

func gatherTapIntervals(md *MapData) {
	var intervals []int
	var previousTime int
	seenFirst := false

	for _, ho := range md.Map.HitObjects {
		if ho.Type&int(osu.HitSpinner) != 0 {
			continue
		}
		if ho.Type&int(osu.HitNormal) == 0 && ho.Type&int(osu.HitSlider) == 0 {
			continue
		}

		if seenFirst {
			intervals = append(intervals, ho.Time-previousTime)
		} else {
			seenFirst = true
		}
		previousTime = ho.Time
	}

	md.PressIntervals = intervals
}

// CalculateTapStrains ports the original C++ tap-strain calculation from
// strains.cpp. It turns the press intervals into the per-note strain curve
// that stamina uses for the final score.
func CalculateTapStrains(md *MapData, vars *Vars) {
	md.TapStrains = make([]float64, 0, len(md.PressIntervals))
	oldBonus := 0.0

	for i, interval := range md.PressIntervals {
		strain := 0.0
		if i == 0 {
			if interval < int(vars.Get("Stamina", "LargestInterval")) {
				exponent := math.Pow(float64(interval), vars.Get("Stamina", "Pow")) * vars.Get("Stamina", "Mult")
				strain = vars.Get("Stamina", "Scale") / math.Pow(float64(interval), exponent)
			}
			md.TapStrains = append(md.TapStrains, strain)
		} else {
			if interval >= int(vars.Get("Stamina", "LargestInterval")) {
				strain = oldBonus * vars.Get("Stamina", "DecayMax")
			} else {
				if interval <= 15 {
					md.TapStrains = append(md.TapStrains, 0)
					oldBonus = 0
					continue
				}
				exponent := math.Pow(float64(interval), vars.Get("Stamina", "Pow")) * vars.Get("Stamina", "Mult")
				strain = vars.Get("Stamina", "Scale") / math.Pow(float64(interval), exponent)
				strain += oldBonus * vars.Get("Stamina", "Decay")
			}
			md.TapStrains = append(md.TapStrains, strain)
		}
		oldBonus = strain
	}
}

func calculateAnglesAndBonuses(md *MapData) {
	n := len(md.AimPoints)

	// Distances считаем всегда
	md.Distances = make([]float64, n)
	if n > 0 {
		md.Distances[0] = 0
	}

	for i := 1; i < n; i++ {
		md.Distances[i] = md.AimPoints[i].Pos.DistanceFrom(md.AimPoints[i-1].Pos)
	}

	// Углы возможны только если есть 3 точки
	if n < 3 {
		md.Angles = make([]float64, 0)
		md.AngleBonuses = make([]float64, 0)
		return
	}

	md.Angles = make([]float64, n-2)
	for i := 0; i+2 < n; i++ {
		md.Angles[i] = getDirAngle(
			md.AimPoints[i].Pos,
			md.AimPoints[i+1].Pos,
			md.AimPoints[i+2].Pos,
		)
	}

	md.AngleBonuses = make([]float64, len(md.Angles))
	oldAngle := md.Angles[0] - 2*md.Angles[0]

	for i, angle := range md.Angles {
		bonus := 0.0
		absAngle := math.Abs(angle)

		if sign(angle) == sign(oldAngle) {
			if absAngle < 90 {
				bonus = math.Sin(degToRad(absAngle)*0.784 + 0.339837)
			} else {
				bonus = math.Sin(degToRad(absAngle))
			}
		} else {
			if absAngle < 90 {
				bonus = math.Sin(degToRad(absAngle)*0.536 + 0.72972)
			} else {
				bonus = math.Sin(degToRad(absAngle)) / 2
			}
		}

		md.AngleBonuses[i] = bonus
		oldAngle = angle
	}

	md.Distances = make([]float64, n)
	md.Distances[0] = 0
	for i := 1; i < n; i++ {
		md.Distances[i] = md.AimPoints[i].Pos.DistanceFrom(md.AimPoints[i-1].Pos)
	}
}
