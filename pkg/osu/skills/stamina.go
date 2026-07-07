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

	strain := 0.0
	md.TapStrains = make([]float64, n)

	for i := 0; i < n; i++ {
		interval := float64(md.PressIntervals[i])

		// Вычисляем difficulty для текущего интервала
		diff := calculateStaminaDifficulty(interval)

		// Decay
		decay := 0.0
		if i > 0 {
			prevInterval := float64(md.PressIntervals[i-1])
			decay = math.Pow(staminaStrainDecayBase, prevInterval/1000.0)
		}

		strain = strain*decay + diff
		md.TapStrains[i] = strain
	}

	// Находим пики strain
	topWeights := getPeakVals(md.TapStrains)

	// Финальная агрегация
	md.Skills.Stamina = getWeightedValue2(topWeights, vars.Get("Stamina", "Weighting"))
	md.Skills.Stamina = vars.Get("Stamina", "TotalMult") * math.Pow(md.Skills.Stamina, vars.Get("Stamina", "TotalPow"))
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
