package skills

import (
	"math"
)

// CalculateStamina портирует stamina.cpp's CalculateStamina.
// Требует calculatePressIntervals (заполняется в calculateTapStrains).
func CalculateStamina(md *MapData, vars *Vars) {
	md.Skills.Stamina = vars.Get("Stamina", "TotalMult") *
		math.Pow(MaxFloatSliceValue(md.TapStrains), vars.Get("Stamina", "TotalPow"))
}

func MaxFloatSliceValue(x []float64) float64 {
	if len(x) < 1 {
		return 0
	}
	m := x[0]
	for i := 1; i < len(x); i++ {
		m = max(m, x[i])
	}
	return m
}
