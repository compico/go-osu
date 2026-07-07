package skills

import (
	"math"
)

// Константы из precision.cpp
const (
	precisionStrainDecayBase = 0.3
	precisionMinDistance     = 30.0
	precisionMaxDistance     = 300.0
	precisionMinAngle        = 15.0
	precisionMaxAngle        = 165.0
)

// CalculatePrecision портирует precision.cpp's CalculatePrecision.
// Оценивает способность попадать по маленьким кругам и выполнять точные движения.
// Требует md.AimPoints, md.Distances, md.Angles (заполняются в PrepareMapData).
func CalculatePrecision(md *MapData, vars *Vars) {
	n := len(md.AimPoints)
	if n == 0 {
		md.Skills.Precision = 0
		return
	}

	cs := md.Map.CircleSize
	circleRadius := cs2px(cs)

	strain := 0.0
	md.PrecisionStrains = make([]float64, n)

	for i := 0; i < n; i++ {
		diff := calculatePrecisionDifficulty(md, i, circleRadius)

		decay := 0.0
		if i > 0 {
			prevInterval := float64(md.AimPoints[i].Time - md.AimPoints[i-1].Time)
			decay = math.Pow(precisionStrainDecayBase, prevInterval/1000.0)
		}

		strain = strain*decay + diff
		md.PrecisionStrains[i] = strain
	}

	topWeights := getPeakVals(md.PrecisionStrains)

	md.Skills.Precision = getWeightedValue2(topWeights, vars.Get("Precision", "Weighting"))
	md.Skills.Precision = vars.Get("Precision", "TotalMult") * math.Pow(md.Skills.Precision, vars.Get("Precision", "TotalPow"))
}

// calculatePrecisionDifficulty оценивает сложность попадания для конкретной точки.
func calculatePrecisionDifficulty(md *MapData, index int, circleRadius float64) float64 {
	if index == 0 {
		return 0
	}

	diff := 0.0

	// 1. Сложность размера круга (меньше круг = сложнее)
	// CS 1-10 → radius 50-4.5 пикселей
	// Нормализуем к диапазону [0, 1], где 1 = самый маленький круг (CS 10)
	sizeDifficulty := 1.0 - (circleRadius-4.5)/(50.0-4.5)
	sizeDifficulty = clampVal(sizeDifficulty, 0.0, 1.0)

	// 2. Сложность расстояния (длинные прыжки сложнее)
	distance := md.Distances[index]
	if distance > 0 {
		normalizedDist := clampVal(distance, precisionMinDistance, precisionMaxDistance)
		normalizedDist = (normalizedDist - precisionMinDistance) / (precisionMaxDistance - precisionMinDistance)
		distanceDifficulty := math.Pow(normalizedDist, 1.5)
		diff += distanceDifficulty * 2.0
	}

	// 3. Сложность угла (резкие углы сложнее)
	if index > 0 && index < len(md.AimPoints)-1 && index-1 < len(md.Angles) {
		angle := math.Abs(md.Angles[index-1])
		if angle > 0 {
			normalizedAngle := clampVal(angle, precisionMinAngle, precisionMaxAngle)
			normalizedAngle = (normalizedAngle - precisionMinAngle) / (precisionMaxAngle - precisionMinAngle)
			// Острые углы (близкие к 180°) сложнее
			angleDifficulty := math.Pow(normalizedAngle, 1.3)
			diff += angleDifficulty * 1.5
		}
	}

	// 4. Сложность скорости (быстрые переходы сложнее)
	interval := float64(md.AimPoints[index].Time - md.AimPoints[index-1].Time)
	if interval > 0 && distance > 0 {
		velocity := distance / interval
		// Нормализуем скорость (типичный диапазон 0.1 - 2.0 пикселей/мс)
		normalizedVelocity := clampVal(velocity, 0.1, 2.0)
		normalizedVelocity = (normalizedVelocity - 0.1) / (2.0 - 0.1)
		speedDifficulty := math.Pow(normalizedVelocity, 1.8)
		diff += speedDifficulty * 2.5
	}

	// Применяем множитель размера круга
	diff *= (1.0 + sizeDifficulty*2.0)

	return diff
}
