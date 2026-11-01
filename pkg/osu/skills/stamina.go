package skills

import (
	"math"
)

// CalculateStamina портирует stamina.cpp's CalculateStamina.
// Требует calculatePressIntervals (заполняется в calculateTapStrains).
func CalculateStamina(md *MapData, vars *Vars) {
	md.calculateStamina(vars)
}

func (md *MapData) calculateStamina(vars *Vars) {
	md.Skills.Stamina = staminaScale(MaxFloatSliceValue(md.TapStrains), vars)
}

// staminaScale converts a tap strain to the stamina skill scale. The skill
// value is the scaled maximum strain, so the same conversion applied to every
// strain gives a curve whose peak equals the skill.
func staminaScale(strain float64, vars *Vars) float64 {
	return vars.Stamina.TotalMult * math.Pow(strain, vars.Stamina.TotalPow)
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
