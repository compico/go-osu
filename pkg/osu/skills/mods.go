package skills

import (
	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

// ApplyMods возвращает глубокую копию Beatmap с параметрами, изменёнными под моды.
// Модифицирует AR, OD, HP, CS, а также тайминги и координаты (для HR).
// Оригинальный Beatmap не изменяется.
//
// Порядок применения: HR/EZ → DT/HT (как в оригинальной osu!).
func ApplyMods(original *osu.Beatmap, mods osu.Mod) *osu.Beatmap {
	if mods == 0 {
		return original // оптимизация: без модов — возвращаем оригинал
	}

	bm := *original

	// --- Глубокое копирование TimingPoints ---
	bm.TimingPoints = make([]osu.TimingPoint, len(original.TimingPoints))
	copy(bm.TimingPoints, original.TimingPoints)
	// Восстанавливаем ссылки PreviousTimingPoint на новый слайс
	for i := range bm.TimingPoints {
		if i > 0 {
			bm.TimingPoints[i].PreviousTimingPoint = &bm.TimingPoints[i-1]
		}
	}

	// --- Глубокое копирование HitObjects ---
	bm.HitObjects = make([]osu.HitObject, len(original.HitObjects))
	for i := range original.HitObjects {
		bm.HitObjects[i] = original.HitObjects[i]
		if len(original.HitObjects[i].Curves) > 0 {
			bm.HitObjects[i].Curves = make([]vector2d.Vector2dd, len(original.HitObjects[i].Curves))
			copy(bm.HitObjects[i].Curves, original.HitObjects[i].Curves)
		}
		if len(original.HitObjects[i].LerpPoints) > 0 {
			bm.HitObjects[i].LerpPoints = make([]vector2d.Vector2dd, len(original.HitObjects[i].LerpPoints))
			copy(bm.HitObjects[i].LerpPoints, original.HitObjects[i].LerpPoints)
		}
		if len(original.HitObjects[i].RepeatTimes) > 0 {
			bm.HitObjects[i].RepeatTimes = make([]int, len(original.HitObjects[i].RepeatTimes))
			copy(bm.HitObjects[i].RepeatTimes, original.HitObjects[i].RepeatTimes)
		}
		if len(original.HitObjects[i].Ticks) > 0 {
			bm.HitObjects[i].Ticks = make([]int, len(original.HitObjects[i].Ticks))
			copy(bm.HitObjects[i].Ticks, original.HitObjects[i].Ticks)
		}
	}

	// --- HR: +40% AR/OD/HP, +30% CS, инверсия Y ---
	if mods&osu.HR != 0 {
		bm.ApproachRate = clampF(bm.ApproachRate*1.4, 0, 10)
		bm.OverallDifficulty = clampF(bm.OverallDifficulty*1.4, 0, 10)
		bm.HPDrainRate = clampF(bm.HPDrainRate*1.4, 0, 10)
		bm.CircleSize = clampF(bm.CircleSize*1.3, 0, 10)

		// Инвертируем Y координаты (playfield 512×384 в osu!pixels)
		for i := range bm.HitObjects {
			bm.HitObjects[i].Pos.Y = 384 - bm.HitObjects[i].Pos.Y
			for j := range bm.HitObjects[i].Curves {
				bm.HitObjects[i].Curves[j].Y = 384 - bm.HitObjects[i].Curves[j].Y
			}
			// EndPoint будет пересчитан в PrepareMapData на основе новых якорей
		}
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
		bm.ApproachRate = applySpeedMod(bm.ApproachRate, 1.5)
		bm.OverallDifficulty = applySpeedMod(bm.OverallDifficulty, 1.5)
		bm.HPDrainRate = applySpeedMod(bm.HPDrainRate, 1.5)

		for i := range bm.TimingPoints {
			bm.TimingPoints[i].Time /= 1.5
			if bm.TimingPoints[i].Uninherited {
				bm.TimingPoints[i].BeatLength /= 1.5
			}
		}
		for i := range bm.HitObjects {
			bm.HitObjects[i].Time = int(float64(bm.HitObjects[i].Time) / 1.5)
			bm.HitObjects[i].EndTime = int(float64(bm.HitObjects[i].EndTime) / 1.5)
		}
	}

	// --- HT: ×0.75 скорость ---
	if mods&osu.HT != 0 {
		bm.ApproachRate = applySpeedMod(bm.ApproachRate, 0.75)
		bm.OverallDifficulty = applySpeedMod(bm.OverallDifficulty, 0.75)
		bm.HPDrainRate = applySpeedMod(bm.HPDrainRate, 0.75)

		for i := range bm.TimingPoints {
			bm.TimingPoints[i].Time /= 0.75
			if bm.TimingPoints[i].Uninherited {
				bm.TimingPoints[i].BeatLength /= 0.75
			}
		}
		for i := range bm.HitObjects {
			bm.HitObjects[i].Time = int(float64(bm.HitObjects[i].Time) / 0.75)
			bm.HitObjects[i].EndTime = int(float64(bm.HitObjects[i].EndTime) / 0.75)
		}
	}

	return &bm
}

// applySpeedMod применяет формулу osu! для DT/HT к параметрам AR/OD/HP.
// При value ≤ 5: new = value × multiplier
// При value > 5: new = (value − 5) × multiplier + 5
func applySpeedMod(value, multiplier float64) float64 {
	if value <= 5 {
		value = value * multiplier
	} else {
		value = (value-5)*multiplier + 5
	}
	return clampF(value, 0, 10)
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
