package skills

import (
	"math"
)

// Константы из accuracy.cpp
const (
	accuracyStrainDecayBase = 0.3
	accuracyMinDistance     = 20.0
	accuracyMaxDistance     = 250.0
	accuracyMinAngle        = 10.0
	accuracyMaxAngle        = 170.0
)

// CalculateAccuracy портирует accuracy.cpp's CalculateAccuracy.
// Оценивает способность точно попадать по объектам с учетом CS, OD, расстояний и углов.
// Требует md.AimPoints, md.Distances, md.Angles (заполняются в PrepareMapData).
func CalculateAccuracy(md *MapData, vars *Vars) {
	n := len(md.AimPoints)
	if n == 0 {
		md.Skills.Accuracy = 0
		return
	}

	cs := md.Map.CircleSize
	od := md.Map.OverallDifficulty
	circleRadius := cs2px(cs)

	strain := 0.0
	md.AccuracyStrains = make([]float64, n)

	for i := 0; i < n; i++ {
		diff := calculateAccuracyDifficulty(md, i, circleRadius, od)

		decay := 0.0
		if i > 0 {
			prevInterval := float64(md.AimPoints[i].Time - md.AimPoints[i-1].Time)
			decay = math.Pow(accuracyStrainDecayBase, prevInterval/1000.0)
		}

		strain = strain*decay + diff
		md.AccuracyStrains[i] = strain
	}

	topWeights := getPeakVals(md.AccuracyStrains)

	md.Skills.Accuracy = getWeightedValue2(topWeights, vars.Get("Accuracy", "Weighting"))
	md.Skills.Accuracy = vars.Get("Accuracy", "TotalMult") * math.Pow(md.Skills.Accuracy, vars.Get("Accuracy", "TotalPow"))
}

// calculateAccuracyDifficulty оценивает сложность точного попадания для конкретной точки.
func calculateAccuracyDifficulty(md *MapData, index int, circleRadius, od float64) float64 {
	if index == 0 {
		return 0
	}

	diff := 0.0

	// 1. Сложность размера круга (меньше круг = сложнее попасть)
	// CS 1-10 → radius 50-4.5 пикселей
	// Нормализуем к диапазону [0, 1], где 1 = самый маленький круг (CS 10)
	sizeDifficulty := 1.0 - (circleRadius-4.5)/(50.0-4.5)
	sizeDifficulty = clampVal(sizeDifficulty, 0.0, 1.0)

	// 2. Сложность OD (меньше окно попадания = сложнее)
	// OD 0-10 → window 80-20ms (для hit 300)
	// Нормализуем к диапазону [0, 1], где 1 = самый высокий OD
	odDifficulty := (od - 0.0) / (10.0 - 0.0)
	odDifficulty = clampVal(odDifficulty, 0.0, 1.0)

	// 3. Сложность расстояния (длинные прыжки требуют большей точности)
	distance := md.Distances[index]
	if distance > 0 {
		normalizedDist := clampVal(distance, accuracyMinDistance, accuracyMaxDistance)
		normalizedDist = (normalizedDist - accuracyMinDistance) / (accuracyMaxDistance - accuracyMinDistance)
		distanceDifficulty := math.Pow(normalizedDist, 1.6)
		diff += distanceDifficulty * 2.0
	}

	// 4. Сложность угла (резкие углы сложнее)
	if index-1 < len(md.Angles) {
		angle := math.Abs(md.Angles[index-1])
		if angle > 0 {
			normalizedAngle := clampVal(angle, accuracyMinAngle, accuracyMaxAngle)
			normalizedAngle = (normalizedAngle - accuracyMinAngle) / (accuracyMaxAngle - accuracyMinAngle)
			// Острые углы (близкие к 180°) сложнее
			angleDifficulty := math.Pow(normalizedAngle, 1.4)
			diff += angleDifficulty * 1.8
		}
	}

	// 5. Сложность скорости (быстрые переходы требуют большей точности)
	interval := float64(md.AimPoints[index].Time - md.AimPoints[index-1].Time)
	if interval > 0 && distance > 0 {
		velocity := distance / interval
		// Нормализуем скорость (типичный диапазон 0.05 - 1.5 пикселей/мс)
		normalizedVelocity := clampVal(velocity, 0.05, 1.5)
		normalizedVelocity = (normalizedVelocity - 0.05) / (1.5 - 0.05)
		speedDifficulty := math.Pow(normalizedVelocity, 2.0)
		diff += speedDifficulty * 2.5
	}

	// Применяем множители размера круга и OD
	diff *= (1.0 + sizeDifficulty*2.5)
	diff *= (1.0 + odDifficulty*2.0)

	return diff
}
