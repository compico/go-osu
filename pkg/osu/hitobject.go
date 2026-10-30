package osu

import "slices"

// Clone returns an independent copy of the hit object, including its
// precomputed curve points, repeat times, and slider ticks.
func (ho HitObject) Clone() HitObject {
	ho.Curves = slices.Clone(ho.Curves)
	ho.LerpPoints = slices.Clone(ho.LerpPoints)
	ho.RepeatTimes = slices.Clone(ho.RepeatTimes)
	ho.Ticks = slices.Clone(ho.Ticks)
	return ho
}
