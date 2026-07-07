package skills

import (
	"math"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

// PrepareMapData заполняет все необходимые поля MapData для расчёта скиллов.
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

	for _, ho := range md.Map.HitObjects {
		if ho.Type&int(osu.HitSpinner) != 0 {
			md.Spinners++
			continue
		}

		// 1. Начало объекта (Circle или Slider)
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

		// 2. Если это слайдер, добавляем реверсы и конец
		if ho.Type&int(osu.HitSlider) != 0 {
			// Собираем все контрольные точки (начало + якоря)
			points := append([]vector2d.Vector2dd{ho.Pos}, ho.Curves...)

			// Создаем путь слайдера с использованием новой геометрии
			sp := NewSliderPath(points, ho.CurveType, ho.PixelLength)

			duration := calculateSliderDuration(&ho, md.Map)
			ho.EndTime = ho.Time + duration
			ho.RepeatTimes = calculateRepeatTimes(&ho, duration)

			// Добавляем точки реверсов
			for i, t := range ho.RepeatTimes {
				var revPos vector2d.Vector2dd
				// В osu! реверсы чередуются:
				// 1-й реверс (индекс 0) в конце слайдера (distance = PixelLength)
				// 2-й реверс (индекс 1) в начале слайдера (distance = 0)
				// 3-й реверс (индекс 2) снова в конце, и так далее.
				if i%2 == 0 {
					revPos = sp.PositionAt(ho.PixelLength)
				} else {
					revPos = sp.PositionAt(0)
				}

				aimPoints = append(aimPoints, AimPoint{
					Time: t,
					Pos:  revPos,
					Type: AimPointSliderReverse,
				})
				timeMapper[t] = len(aimPoints) - 1
			}

			// Добавляем конец слайдера
			endPos := sp.PositionAt(ho.PixelLength)
			aimPoints = append(aimPoints, AimPoint{
				Time: ho.EndTime,
				Pos:  endPos,
				Type: AimPointSliderEnd,
			})
			timeMapper[ho.EndTime] = len(aimPoints) - 1

			// Сохраняем точный EndPoint в HitObject для возможного использования в других скиллах
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

func calculateAnglesAndDistances(md *MapData) {
	n := len(md.AimPoints)
	if n == 0 {
		return
	}

	md.Angles = make([]float64, n)
	md.Distances = make([]float64, n)

	md.Distances[0] = 0.0
	md.Angles[0] = 0.0

	for i := 1; i < n; i++ {
		prevPos := md.AimPoints[i-1].Pos
		currPos := md.AimPoints[i].Pos
		md.Distances[i] = currPos.DistanceFrom(prevPos)

		if i+1 < n {
			nextPos := md.AimPoints[i+1].Pos
			md.Angles[i] = getDirAngle(prevPos, currPos, nextPos)
		} else {
			md.Angles[i] = 0.0
		}
	}
}

func gatherTapIntervals(md *MapData) {
	if len(md.AimPoints) < 2 {
		return
	}

	md.PressIntervals = make([]int, len(md.AimPoints))
	md.PressIntervals[0] = 0

	for i := 1; i < len(md.AimPoints); i++ {
		md.PressIntervals[i] = md.AimPoints[i].Time - md.AimPoints[i-1].Time
	}
}

func calculateAnglesAndBonuses(md *MapData) {
	n := len(md.AimPoints)
	if n < 3 {
		md.Angles = make([]float64, 0)
		md.AngleBonuses = make([]float64, 0)
		return
	}

	md.Angles = make([]float64, n-2)
	for i := 0; i+2 < n; i++ {
		angle := getDirAngle(md.AimPoints[i].Pos, md.AimPoints[i+1].Pos, md.AimPoints[i+2].Pos)
		md.Angles[i] = angle
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
