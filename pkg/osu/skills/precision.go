package skills

import (
	"math"
)

// CalculatePrecision портирует precision.cpp's CalculatePrecision.
// Рассчитывает точность на основе agility и размера круга.
// Требует предварительного расчёта agility в agility.
func CalculatePrecision(md *MapData, vars *Vars) {
	md.calculatePrecision(vars)
}

func (md *MapData) calculatePrecision(vars *Vars) {
	md.Skills.Precision = precisionFromAgility(md.Skills.Agility, md.Map.CircleSize, vars)
}

// precisionFromAgility is the precision skill of a map with the given agility
// and circle size. It grows with agility, so applying it to every point of the
// agility curve gives a curve whose peak equals the skill.
func precisionFromAgility(agility, cs float64, vars *Vars) float64 {
	debugf("[PRECISION] agilityAtCallTime=%v cs=%v\n", agility, cs)

	// Масштабируем agility
	var scaledAgility float64
	agilityLimit := vars.Precision.AgilityLimit
	if agility > agilityLimit {
		scaledAgility = 1.0
	}

	agilityPow := vars.Precision.AgilityPow
	agilitySubtract := vars.Precision.AgilitySubtract
	scaledAgility = math.Pow(agility+1, agilityPow) - agilitySubtract

	debugf("[PRECISION] agilityLimit=%v agilityPow=%v agilitySubtract=%v scaledAgility=%v\n",
		agilityLimit, agilityPow, agilitySubtract, scaledAgility)

	// precision = scaledAgility * cs
	precision := scaledAgility * cs

	debugf("[PRECISION] precision(before totalMult/Pow)=%v\n", precision)

	// Применяем финальный множитель
	totalMult := vars.Precision.TotalMult
	totalPow := vars.Precision.TotalPow

	result := totalMult * math.Pow(precision, totalPow)

	debugf("[PRECISION] totalMult=%v totalPow=%v final=%v\n", totalMult, totalPow, result)
	return result
}
