package osu

import "slices"

// HitObjectType flags
type HitObjectType int

const (
	HitNormal         HitObjectType = 1
	HitSlider         HitObjectType = 2
	HitNewCombo       HitObjectType = 4
	HitNormalNewCombo HitObjectType = 5
	HitSliderNewCombo HitObjectType = 6
	HitSpinner        HitObjectType = 8
	HitColourHax      HitObjectType = 112
	HitHold           HitObjectType = 128
	HitManiaLong      HitObjectType = 128
)

func (ho HitObjectType) IsHitNormal() bool {
	return ho&HitNormal != 0
}

func (ho HitObjectType) IsHitSlider() bool {
	return ho&HitSlider != 0
}

func (ho HitObjectType) IsHitNewCombo() bool {
	return ho&HitNewCombo != 0
}

func (ho HitObjectType) IsHitNormalNewCombo() bool {
	return ho&HitNormalNewCombo != 0
}

func (ho HitObjectType) IsHitSliderNewCombo() bool {
	return ho&HitSliderNewCombo != 0
}

func (ho HitObjectType) IsHitSpinner() bool {
	return ho&HitSpinner != 0
}

func (ho HitObjectType) IsHitColourHax() bool {
	return ho&HitColourHax != 0
}

func (ho HitObjectType) IsHitHold() bool {
	return ho&HitHold != 0
}

func (ho HitObjectType) IsHitManiaLong() bool {
	return ho&HitManiaLong != 0
}

// Clone returns an independent copy of the hit object, including its
// precomputed curve points, repeat times, and slider ticks.
func (ho HitObject) Clone() HitObject {
	ho.Curves = slices.Clone(ho.Curves)
	ho.LerpPoints = slices.Clone(ho.LerpPoints)
	ho.RepeatTimes = slices.Clone(ho.RepeatTimes)
	ho.Ticks = slices.Clone(ho.Ticks)
	return ho
}
