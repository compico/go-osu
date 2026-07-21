package skills

import (
	"math"
	"slices"
)

// CalculateStamina портирует stamina.cpp's CalculateStamina.
// Требует calculatePressIntervals (заполняется в calculateTapStrains).
func CalculateStamina(md *MapData, vars *Vars) {
	md.Skills.Stamina = vars.Get("Stamina", "TotalMult") *
		math.Pow(slices.Max(md.TapStrains), vars.Get("Stamina", "TotalPow"))
}
