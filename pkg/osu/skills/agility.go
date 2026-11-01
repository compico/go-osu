package skills

import (
	"math"
)

// CalculateAgility агрегирует AimStrains в финальный скилл Agility.
func CalculateAgility(md *MapData, vars *Vars) {
	md.calculateAgility(vars)
}

func (md *MapData) calculateAgility(vars *Vars) {
	maxStrain := 0.0
	maxIndex := 0

	for i, v := range md.AimStrains {
		if v > maxStrain {
			maxStrain = v
			maxIndex = i
		}
	}

	time := md.AimPoints[maxIndex].Time

	md.Skills.Agility = maxStrain
	debugf("[AGILITY] aimStrains.size()=%v max=%v at index=%v time=%v\n", len(md.AimStrains), maxStrain, maxIndex, time)

	topWeights := getPeakVals(md.AimStrains)
	debugf("[AGILITY] topWeights.size()=%v", len(topWeights))
	for i, t := range topWeights {
		debugf(" [%d]=%v", i, t)
	}
	debugf("\n")

	md.Skills.Agility = getWeightedValue2(
		topWeights,
		vars.Agility.Weighting,
	)

	debugf(
		"[AGILITY] weighted=%v Weighting=%v TotalMult=%v TotalPow=%v\n",
		md.Skills.Agility,
		vars.Agility.Weighting,
		vars.Agility.TotalMult,
		vars.Agility.TotalPow,
	)

	md.Skills.Agility = agilityScale(md.Skills.Agility, vars)

	debugf("[AGILITY] final=%v\n", md.Skills.Agility)
}

// agilityScale applies the final transformation of the agility skill to an
// aggregated strain.
func agilityScale(strain float64, vars *Vars) float64 {
	return vars.Agility.TotalMult * math.Pow(strain, vars.Agility.TotalPow)
}
