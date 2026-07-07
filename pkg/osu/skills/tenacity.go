package skills

import (
	"math"
)

// Константы из tenacity.cpp
const (
	tenacityStreamThreshold = 125.0 // Максимальный интервал для стрима (ms)
	tenacityBurstThreshold  = 75.0  // Максимальный интервал для батрста (ms)
	tenacityMinStreamLength = 3     // Минимальная длина стрима (количество интервалов)
	tenacityStrainDecayBase = 0.3
)

// CalculateTenacity портирует tenacity.cpp's CalculateTenacity.
// Требует md.PressIntervals (заполняется в gatherTapIntervals).
func CalculateTenacity(md *MapData, vars *Vars) {
	n := len(md.PressIntervals)
	if n == 0 {
		md.Skills.Tenacity = 0
		return
	}

	strain := 0.0
	md.TenacityStrains = make([]float64, n)

	for i := 0; i < n; i++ {
		diff := calculateTenacityDifficulty(md.PressIntervals, i)

		decay := 0.0
		if i > 0 {
			prevInterval := float64(md.PressIntervals[i-1])
			decay = math.Pow(tenacityStrainDecayBase, prevInterval/1000.0)
		}

		strain = strain*decay + diff
		md.TenacityStrains[i] = strain
	}

	topWeights := getPeakVals(md.TenacityStrains)

	md.Skills.Tenacity = getWeightedValue2(topWeights, vars.Get("Tenacity", "Weighting"))
	md.Skills.Tenacity = vars.Get("Tenacity", "TotalMult") * math.Pow(md.Skills.Tenacity, vars.Get("Tenacity", "TotalPow"))
}

// calculateTenacityDifficulty оценивает сложность для конкретной точки,
// анализируя длину и скорость текущего стрима/батрста, глядя назад.
// Это точный порт логики из tenacity.cpp.
func calculateTenacityDifficulty(pressIntervals []int, index int) float64 {
	diff := 0.0

	// 1. Анализ стрима (Stream)
	// Стрим - это последовательность интервалов <= tenacityStreamThreshold
	streamLength := 0
	streamIntervalSum := 0

	for i := index; i > 0; i-- {
		interval := pressIntervals[i]
		if float64(interval) <= tenacityStreamThreshold {
			streamLength++
			streamIntervalSum += interval
		} else {
			break // Прерываем, как только встретили медленный интервал
		}
	}

	if streamLength >= tenacityMinStreamLength {
		avgStreamInterval := float64(streamIntervalSum) / float64(streamLength)
		// Длинные стримы сложнее (экспоненциальный рост)
		lengthBonus := math.Pow(float64(streamLength), 1.2)
		// Быстрые стримы сложнее
		speedBonus := math.Pow(tenacityStreamThreshold/avgStreamInterval, 1.5)
		diff += lengthBonus * speedBonus
	}

	// 2. Анализ батрста (Burst)
	// Батрст - это очень быстрая последовательность (интервал <= tenacityBurstThreshold)
	// Обычно короче стрима, но требует взрывной скорости
	burstLength := 0
	burstIntervalSum := 0

	for i := index; i > 0; i-- {
		interval := pressIntervals[i]
		if float64(interval) <= tenacityBurstThreshold {
			burstLength++
			burstIntervalSum += interval
		} else {
			break
		}
	}

	if burstLength >= 2 { // Батрсты могут быть короче стримов (например, 2-3 ноты)
		avgBurstInterval := float64(burstIntervalSum) / float64(burstLength)
		// Батрсты дают бонус за экстремальную скорость
		burstSpeedBonus := math.Pow(tenacityBurstThreshold/avgBurstInterval, 2.5)
		// Небольшой бонус за длину батрста
		burstLengthBonus := math.Pow(float64(burstLength), 0.8)
		diff += burstSpeedBonus * burstLengthBonus * 0.5 // Множитель для балансировки
	}

	return diff
}
