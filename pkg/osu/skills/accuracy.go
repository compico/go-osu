package skills

import (
	"math"

	"github.com/chewxy/math32"
	"github.com/compico/go-osu/pkg/osu"
)

// CalculateAccuracy портирует accuracy.cpp's CalculateAccuracy.
// Рассчитывает точность на основе stamina и параметров карты.
// Требует предварительного расчёта stamina в md.Skills.Stamina.
func CalculateAccuracy(md *MapData, vars *Vars) {
	// Считаем только circles (Normal hit objects, не Sliders)
	var circles float32 = 0
	for _, obj := range md.Map.HitObjects {
		// Проверяем, есть ли флаг HitNormal и нет флага HitSlider
		if (obj.Type.IsHitNormal()) && (!obj.Type.IsHitSlider()) {
			circles++
		}
	}

	// SS_UR = (5 * sqrt(2) * OD2ms(od)) / erfInv(pow(0.1, 1.0/circles))
	invCert := math32.Pow(0.1, 1.0/circles)
	ssUr := (5.0 * math32.Sqrt(2.0) * odToMs(float32(md.Map.OverallDifficulty))) / erfInv(invCert)

	// Применяем моды
	if md.HasMod(osu.DT) {
		ssUr /= 1.5
	} else if md.HasMod(osu.HT) {
		ssUr /= 0.75
	}

	// accuracy = stamina^VerScale / SS_UR
	verScale := vars.Get("Accuracy", "VerScale")
	accuracy := math.Pow(md.Skills.Stamina, verScale) / float64(ssUr)

	// Применяем финальный множитель
	totalMult := vars.Get("Accuracy", "TotalMult")
	totalPow := vars.Get("Accuracy", "TotalPow")
	md.Skills.Accuracy = totalMult * math.Pow(accuracy, totalPow)
}

// odToMs converts OD value to milliseconds.
// OD2ms from utils.cpp: -6.0 * od + 79.5
func odToMs(od float32) float32 {
	return -6.0*od + 79.5
}
