package skills

import (
	"math"
)

// CalculateAimStrains вычисляет страйны для Agility v2.
// Требует md.AimPoints, md.AngleBonuses (заполняются в calculateAnglesAndBonuses).
func CalculateAimStrains(md *MapData, vars *Vars) {
	md.AimStrains = make([]float64, 0, len(md.AimPoints))
	oldStrain := 0.0

	for i := 0; i < len(md.AimPoints); i++ {
		strain := 0.0
		if i > 0 {
			distance := getWeightedAimDistance(md.AimPoints[i].Pos.DistanceFrom(md.AimPoints[i-1].Pos), vars)
			interval := md.AimPoints[i].Time - md.AimPoints[i-1].Time
			time := getWeightedAimTime(float64(interval), vars)

			angleBonus := 1.0
			if i > 1 && i-2 < len(md.AngleBonuses) {
				angleBonus = 1 + (vars.Get("Agility", "AngleMult") * md.AngleBonuses[i-2])
			}

			if time > 0 {
				strain = distance / time * angleBonus
			} else {
				continue
			}

			// Уменьшаем вес для слайдеров
			if md.AimPoints[i].Type == AimPointSliderEnd || md.AimPoints[i-1].Type == AimPointSliderEnd {
				strain *= vars.Get("Agility", "SliderStrainDecay")
			}

			oldStrain -= vars.Get("Agility", "StrainDecay") * float64(interval)
			if oldStrain < 0 {
				oldStrain = 0
			}

			strain += oldStrain
		}
		md.AimStrains = append(md.AimStrains, strain)
		oldStrain = strain
	}
}

// getWeightedAimDistance применяет бонус к расстоянию.
func getWeightedAimDistance(distance float64, vars *Vars) float64 {
	distanceBonus := math.Pow(1+(distance*vars.Get("Agility", "DistMult")), vars.Get("Agility", "DistPow"))
	distanceBonus /= vars.Get("Agility", "DistDivisor")
	return distance * distanceBonus
}

// getWeightedAimTime применяет бонус к времени.
func getWeightedAimTime(time float64, vars *Vars) float64 {
	timeBonus := math.Pow(time*vars.Get("Agility", "TimeMult"), vars.Get("Agility", "TimePow"))
	return time * timeBonus
}

// CalculateAgility агрегирует AimStrains в финальный скилл Agility.
func CalculateAgility(md *MapData, vars *Vars) {
	if len(md.AimStrains) == 0 {
		md.Skills.Agility = 0
		return
	}

	// Находим пики страйнов
	topWeights := getPeakVals(md.AimStrains)

	// Агрегируем
	md.Skills.Agility = getWeightedValue2(topWeights, vars.Get("Agility", "Weighting"))
	md.Skills.Agility = vars.Get("Agility", "TotalMult") * math.Pow(md.Skills.Agility, vars.Get("Agility", "TotalPow"))
}
