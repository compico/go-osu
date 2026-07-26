package skills

import (
	"math"
)

// CalculateAgility агрегирует AimStrains в финальный скилл Agility.
func CalculateAgility(md *MapData, vars *Vars) {
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
		vars.Get("Agility", "Weighting"),
	)

	debugf(
		"[AGILITY] weighted=%v Weighting=%v TotalMult=%v TotalPow=%v\n",
		md.Skills.Agility,
		vars.Get("Agility", "Weighting"),
		vars.Get("Agility", "TotalMult"),
		vars.Get("Agility", "TotalPow"),
	)

	md.Skills.Agility = vars.Get("Agility", "TotalMult") *
		math.Pow(
			md.Skills.Agility,
			vars.Get("Agility", "TotalPow"),
		)

	debugf("[AGILITY] final=%v\n", md.Skills.Agility)
}
