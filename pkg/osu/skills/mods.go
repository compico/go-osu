package skills

import (
	"math"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

// ApplyMods возвращает глубокую копию Beatmap с параметрами, изменёнными под моды.
// Модифицирует AR, OD, HP, CS, а также тайминги и координаты (для HR).
// Оригинальный Beatmap не изменяется.
//
// Порядок применения: HR/EZ → DT/HT (как в оригинальной osu!).
func ApplyMods(original *osu.Beatmap, mods osu.Mod) *osu.Beatmap {
	bm := *original

	// --- Глубокое копирование HitObjects, с фильтрацией спиннеров/hold как в оригинальном C++ парсере ---
	// (C++ FileReader никогда не добавляет Spinner/Hold в bData.hitObjects)
	bm.HitObjects = make([]osu.HitObject, 0, len(original.HitObjects))
	for i := range original.HitObjects {
		src := &original.HitObjects[i]

		if src.Type.IsHitSpinner() || src.Type.IsHitHold() {
			continue
		}

		ho := *src
		if len(src.Curves) > 0 {
			ho.Curves = make([]vector2d.Vector2dd, len(src.Curves))
			copy(ho.Curves, src.Curves)
		}
		if len(src.LerpPoints) > 0 {
			ho.LerpPoints = make([]vector2d.Vector2dd, len(src.LerpPoints))
			copy(ho.LerpPoints, src.LerpPoints)
		}
		if len(src.RepeatTimes) > 0 {
			ho.RepeatTimes = make([]int, len(src.RepeatTimes))
			copy(ho.RepeatTimes, src.RepeatTimes)
		}
		if len(src.Ticks) > 0 {
			ho.Ticks = make([]int, len(src.Ticks))
			copy(ho.Ticks, src.Ticks)
		}
		if ho.Type.IsHitNormal() {
			ho.EndPoint = ho.Pos
		}

		bm.HitObjects = append(bm.HitObjects, ho)
	}

	if mods == 0 {
		return &bm
	}

	// --- Глубокое копирование TimingPoints ---
	bm.TimingPoints = make([]osu.TimingPoint, len(original.TimingPoints))
	copy(bm.TimingPoints, original.TimingPoints)

	// --- HR: +40% AR/OD/HP, +30% CS ---
	if mods&osu.HR != 0 {
		bm.ApproachRate = math.Min(bm.ApproachRate*1.4, 10)
		bm.OverallDifficulty = math.Min(bm.OverallDifficulty*1.4, 10)
		bm.HPDrainRate = math.Min(bm.HPDrainRate*1.4, 10)
		bm.CircleSize = math.Min(bm.CircleSize*1.3, 10)
	}

	// --- EZ: -50% ко всем параметрам ---
	if mods&osu.EZ != 0 {
		bm.ApproachRate = clampF(bm.ApproachRate*0.5, 0, 10)
		bm.OverallDifficulty = clampF(bm.OverallDifficulty*0.5, 0, 10)
		bm.HPDrainRate = clampF(bm.HPDrainRate*0.5, 0, 10)
		bm.CircleSize = clampF(bm.CircleSize*0.5, 0, 10)
	}

	// --- DT: ×1.5 скорость ---
	if mods&osu.DT != 0 {
		const speed = 1.5

		for i := range bm.TimingPoints {
			bm.TimingPoints[i].Time = math.Ceil(bm.TimingPoints[i].Time / speed)
			if bm.TimingPoints[i].Uninherited {
				bm.TimingPoints[i].BeatLength /= speed
			}
		}

		for i := range bm.HitObjects {
			bm.HitObjects[i].Time = int(math.Ceil(float64(bm.HitObjects[i].Time) / speed))
			bm.HitObjects[i].EndTime = int(math.Ceil(float64(bm.HitObjects[i].EndTime) / speed))
		}

		bm.ApproachRate = math.Min(msToAR(int(float64(arToMs(bm.ApproachRate))/speed)), 11)
	}

	// --- HT: ×0.75 скорость ---
	if mods&osu.HT != 0 {
		const speed = 0.75

		for i := range bm.TimingPoints {
			bm.TimingPoints[i].Time = math.Ceil(bm.TimingPoints[i].Time / speed)
			if bm.TimingPoints[i].Uninherited {
				bm.TimingPoints[i].BeatLength /= speed
			}
		}

		for i := range bm.HitObjects {
			bm.HitObjects[i].Time = int(math.Ceil(float64(bm.HitObjects[i].Time) / speed))
			bm.HitObjects[i].EndTime = int(math.Ceil(float64(bm.HitObjects[i].EndTime) / speed))
		}

		bm.ApproachRate = msToAR(int(float64(arToMs(bm.ApproachRate)) / speed))
	}

	return &bm
}

func msToAR(ms int) float64 {
	if ms >= 1200 {
		return (1800 - float64(ms)) / 120
	}
	return (1950 - float64(ms)) / 150
}

func clampF(value, minVal, maxVal float64) float64 {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}
