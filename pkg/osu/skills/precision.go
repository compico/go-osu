package skills

import (
	"math"
)

// CalculatePrecision портирует precision.cpp's CalculatePrecision.
// Рассчитывает точность на основе agility и размера круга.
// Требует предварительного расчёта agility в md.Skills.Agility.
func CalculatePrecision(md *MapData, vars *Vars) {
	cs := md.Map.CircleSize

	// Масштабируем agility
	var scaledAgility float64
	agilityLimit := vars.Get("Precision", "AgilityLimit")
	if md.Skills.Agility > agilityLimit {
		scaledAgility = 1.0
	}

	agilityPow := vars.Get("Precision", "AgilityPow")
	agilitySubtract := vars.Get("Precision", "AgilitySubtract")
	scaledAgility = math.Pow(md.Skills.Agility+1, agilityPow) - agilitySubtract

	// precision = scaledAgility * cs
	precision := scaledAgility * cs

	// Применяем финальный множитель
	totalMult := vars.Get("Precision", "TotalMult")
	totalPow := vars.Get("Precision", "TotalPow")

	md.Skills.Precision = totalMult * math.Pow(precision, totalPow)
}
