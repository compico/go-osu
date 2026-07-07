package skills

import (
	"math"
)

// Константы из reaction.cpp
const (
	reactionStrainDecayBase = 0.3
	reactionMinInterval     = 50.0
	reactionMaxInterval     = 500.0
)

// CalculateReaction портирует reaction.cpp's CalculateReaction.
// Оценивает скорость реакции игрока на основе AR и плотности объектов.
// Требует md.AimPoints (заполняются в gatherAimPoints).
func CalculateReaction(md *MapData, vars *Vars) {
	n := len(md.AimPoints)
	if n == 0 {
		md.Skills.Reaction = 0
		return
	}

	ar := md.Map.ApproachRate
	preemptMs := arToMs(ar)

	strain := 0.0
	md.ReactionStrains = make([]float64, n)

	for i := 0; i < n; i++ {
		diff := calculateReactionDifficulty(md, i, preemptMs)

		decay := 0.0
		if i > 0 {
			prevInterval := float64(md.AimPoints[i].Time - md.AimPoints[i-1].Time)
			decay = math.Pow(reactionStrainDecayBase, prevInterval/1000.0)
		}

		strain = strain*decay + diff
		md.ReactionStrains[i] = strain
	}

	topWeights := getPeakVals(md.ReactionStrains)

	md.Skills.Reaction = getWeightedValue2(topWeights, vars.Get("Reaction", "Weighting"))
	md.Skills.Reaction = vars.Get("Reaction", "TotalMult") * math.Pow(md.Skills.Reaction, vars.Get("Reaction", "TotalPow"))
}

// calculateReactionDifficulty оценивает сложность реакции для конкретной точки.
func calculateReactionDifficulty(md *MapData, index int, preemptMs float64) float64 {
	if index == 0 {
		return 0
	}

	diff := 0.0

	// 1. Сложность AR (меньше времени на реакцию = сложнее)
	// AR 0-10 → preempt 1800-450ms
	// Нормализуем к диапазону [0, 1], где 1 = самый высокий AR
	arDifficulty := 1.0 - (preemptMs-450.0)/(1800.0-450.0)
	arDifficulty = clampVal(arDifficulty, 0.0, 1.0)

	// 2. Сложность интервала (быстрые последовательности сложнее)
	interval := float64(md.AimPoints[index].Time - md.AimPoints[index-1].Time)
	if interval > 0 {
		normalizedInterval := clampVal(interval, reactionMinInterval, reactionMaxInterval)
		normalizedInterval = (reactionMaxInterval - normalizedInterval) / (reactionMaxInterval - reactionMinInterval)
		intervalDifficulty := math.Pow(normalizedInterval, 1.5)
		diff += intervalDifficulty * 2.0
	}

	// 3. Сложность плотности (больше объектов в зоне видимости = сложнее)
	// Считаем количество объектов в окне видимости (preemptMs)
	visibleCount := 0
	for i := index; i >= 0; i-- {
		timeDiff := float64(md.AimPoints[index].Time - md.AimPoints[i].Time)
		if timeDiff > preemptMs {
			break
		}
		visibleCount++
	}

	if visibleCount > 1 {
		// Чем больше объектов видно одновременно, тем сложнее
		densityDifficulty := math.Pow(float64(visibleCount), 1.2)
		diff += densityDifficulty * 1.5
	}

	// Применяем множитель AR
	diff *= (1.0 + arDifficulty*3.0)

	return diff
}
