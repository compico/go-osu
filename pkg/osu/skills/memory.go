package skills

import (
	"math"
)

// Константы из memory.cpp
const (
	memoryStrainDecayBase = 0.3
	memoryWindowSize      = 3000.0
)

// CalculateMemory портирует memory.cpp's CalculateMemory.
// Оценивает сложность запоминания паттернов карты.
// Требует md.AimPoints, md.Angles, md.Distances (заполняются в PrepareMapData).
func CalculateMemory(md *MapData, vars *Vars) {
	n := len(md.AimPoints)
	if n == 0 {
		md.Skills.Memory = 0
		return
	}

	strain := 0.0
	md.MemoryStrains = make([]float64, n)

	for i := 0; i < n; i++ {
		diff := calculateMemoryDifficulty(md, i)

		decay := 0.0
		if i > 0 {
			prevInterval := float64(md.AimPoints[i].Time - md.AimPoints[i-1].Time)
			decay = math.Pow(memoryStrainDecayBase, prevInterval/1000.0)
		}

		strain = strain*decay + diff
		md.MemoryStrains[i] = strain
	}

	topWeights := getPeakVals(md.MemoryStrains)

	md.Skills.Memory = getWeightedValue2(topWeights, vars.Get("Memory", "Weighting"))
	md.Skills.Memory = vars.Get("Memory", "TotalMult") * math.Pow(md.Skills.Memory, vars.Get("Memory", "TotalPow"))
}

// calculateMemoryDifficulty оценивает сложность запоминания для конкретной точки.
// Непредсказуемые паттерны (резкие изменения углов/расстояний) сложнее запомнить.
func calculateMemoryDifficulty(md *MapData, index int) float64 {
	if index < 2 {
		return 0
	}

	diff := 0.0

	// 1. Вариативность углов (резкие изменения направления сложнее запомнить)
	if index-1 < len(md.Angles) && index-2 < len(md.Angles) {
		currentAngle := math.Abs(md.Angles[index-1])
		prevAngle := math.Abs(md.Angles[index-2])

		angleChange := math.Abs(currentAngle - prevAngle)
		if angleChange > 0 {
			// Нормализуем изменение угла (0-180°)
			normalizedChange := clampVal(angleChange, 0, 180) / 180.0
			angleDifficulty := math.Pow(normalizedChange, 1.3)
			diff += angleDifficulty * 2.0
		}
	}

	// 2. Вариативность расстояний (нерегулярные прыжки сложнее запомнить)
	if index < len(md.Distances) && index-1 < len(md.Distances) {
		currentDist := md.Distances[index]
		prevDist := md.Distances[index-1]

		if prevDist > 0 {
			distRatio := currentDist / prevDist
			// Отношение 1.0 = одинаковые расстояния (легко запомнить)
			// Отношение != 1.0 = разные расстояния (сложнее)
			distDeviation := math.Abs(distRatio - 1.0)
			if distDeviation > 0.1 {
				distDifficulty := math.Pow(distDeviation, 1.5)
				diff += distDifficulty * 1.5
			}
		}
	}

	// 3. Сложность паттерна (анализ последних N объектов)
	// Считаем количество уникальных паттернов в окне
	patternComplexity := analyzePatternComplexity(md, index)
	diff += patternComplexity * 2.0

	return diff
}

// analyzePatternComplexity анализирует сложность паттерна в окне времени.
func analyzePatternComplexity(md *MapData, index int) float64 {
	if index < 3 {
		return 0
	}

	windowStart := index - 3
	if windowStart < 0 {
		windowStart = 0
	}

	// Считаем количество изменений направления
	directionChanges := 0
	for i := windowStart + 1; i <= index && i-1 < len(md.Angles); i++ {
		if i >= 2 && i-1 < len(md.Angles) {
			angle := md.Angles[i-1]
			if math.Abs(angle) > 30 { // Значительное изменение направления
				directionChanges++
			}
		}
	}

	// Нормализуем количество изменений
	return float64(directionChanges) / 3.0
}
