package skills

import (
	"math"
)

// Константы из stamina.cpp
const (
	staminaStrainDecayBase = 0.3
	staminaMinInterval     = 25.0
	staminaMaxInterval     = 200.0
	staminaWindowSize      = 3000.0
)

// CalculateStamina портирует stamina.cpp's CalculateStamina.
// Требует md.PressIntervals (заполняется в gatherTapIntervals).
func CalculateStamina(md *MapData, vars *Vars) {
	n := len(md.PressIntervals)
	if n == 0 {
		md.Skills.Stamina = 0
		return
	}

	if len(md.TapStrains) == 0 {
		CalculateTapStrains(md, vars)
	}

	maxStrain := 0.0
	for _, strainValue := range md.TapStrains {
		if strainValue > maxStrain {
			maxStrain = strainValue
		}
	}

	md.Skills.Stamina = vars.Get("Stamina", "TotalMult") * math.Pow(maxStrain, vars.Get("Stamina", "TotalPow"))
}

// calculateStaminaDifficulty оценивает сложность одного интервала.
// Чем короче интервал, тем выше сложность.
func calculateStaminaDifficulty(interval float64) float64 {
	if interval <= 0 {
		return 0
	}

	// Нормализуем интервал к диапазону [0, 1]
	// 25ms = максимальная сложность, 200ms = минимальная
	normalized := clampVal(interval, staminaMinInterval, staminaMaxInterval)
	normalized = (staminaMaxInterval - normalized) / (staminaMaxInterval - staminaMinInterval)

	// Экспоненциальное усиление для очень быстрых интервалов
	return math.Pow(normalized, 2.5) * 10.0
}
