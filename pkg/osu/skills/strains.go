package skills

import "math"

// strains.cpp. It turns the press intervals into the per-note strain curve
// that stamina uses for the final score.
func (md *MapData) calculateTapStrains(vars *Vars) {
	largestInterval := vars.Get("Stamina", "LargestInterval")
	scale := vars.Get("Stamina", "Scale")
	pow := vars.Get("Stamina", "Pow")
	mult := vars.Get("Stamina", "Mult")
	decayMax := vars.Get("Stamina", "DecayMax")
	decay := vars.Get("Stamina", "Decay")

	count := 0
	strain := 0.0
	oldBonus := 0.0

	for _, interval := range md.PressIntervals {
		intervalF := float64(interval)

		if count == 0 {
			if intervalF >= largestInterval {
				strain = 0
			} else {
				strain = scale /
					math.Pow(
						intervalF,
						math.Pow(intervalF, pow)*mult,
					)
			}

			md.TapStrains = append(md.TapStrains, strain)
		} else {
			if intervalF >= largestInterval {
				strain *= decayMax
			} else {
				if interval <= 15 {
					continue
				}

				strain = scale /
					math.Pow(
						intervalF,
						math.Pow(intervalF, pow)*mult,
					)

				strain += oldBonus * decay
			}

			md.TapStrains = append(md.TapStrains, strain)
		}

		oldBonus = strain
		count++
	}
}

// getWeightedAimDistance применяет бонус к расстоянию.
func getWeightedAimDistance(distance, multiplier, power, divisor float64) float64 {
	distanceBonus := math.Pow(1+(distance*multiplier), power)
	distanceBonus /= divisor
	return distance * distanceBonus
}

// getWeightedAimTime применяет бонус к времени.
func getWeightedAimTime(time, multiplier, power float64) float64 {
	timeBonus := math.Pow(time*multiplier, power)
	return time * timeBonus
}

// calculateAimStrains вычисляет страйны для Agility v2.
// Требует md.AimPoints, md.AngleBonuses (заполняются в calculateAnglesAndBonuses).
func (md *MapData) calculateAimStrains(vars *Vars) {
	distanceMultiplier := vars.Get("Agility", "DistMult")
	distancePower := vars.Get("Agility", "DistPow")
	distanceDivisor := vars.Get("Agility", "DistDivisor")
	timeMultiplier := vars.Get("Agility", "TimeMult")
	timePower := vars.Get("Agility", "TimePow")
	angleMultiplier := vars.Get("Agility", "AngleMult")
	sliderStrainDecay := vars.Get("Agility", "SliderStrainDecay")
	strainDecay := vars.Get("Agility", "StrainDecay")

	md.AimStrains = make([]float64, 0, len(md.AimPoints))
	oldStrain := 0.0

	// debug vars
	strainBeforeSlider := 0.0
	oldstrainBeforeDecay := 0.0

	for i := 0; i < len(md.AimPoints); i++ {
		strain := 0.0
		if i > 0 {
			distance := getWeightedAimDistance(
				md.AimPoints[i].Pos.DistanceFrom(md.AimPoints[i-1].Pos),
				distanceMultiplier,
				distancePower,
				distanceDivisor,
			)
			interval := md.AimPoints[i].Time - md.AimPoints[i-1].Time
			time := getWeightedAimTime(float64(interval), timeMultiplier, timePower)

			angleBonus := 1.0
			if i > 1 {
				angleBonus = 1 + (angleMultiplier * md.AngleBonuses[i-2])
			}

			if time > 0 {
				strain = distance / time * angleBonus
			} else {
				debugf("[STRAIN] SKIPPED i=%v (time <=0), interval=%v rawTime=%v\n", i, interval, time)
				continue
			}

			if debug {
				strainBeforeSlider = strain
			}

			// Уменьшаем вес для слайдеров
			if md.AimPoints[i].Type == AimPointSliderEnd || md.AimPoints[i-1].Type == AimPointSliderEnd {
				strain *= sliderStrainDecay
			}

			if debug {
				oldstrainBeforeDecay = oldStrain
			}
			oldStrain -= strainDecay * float64(interval)
			if oldStrain < 0 {
				oldStrain = 0
			}

			strain += oldStrain
			if debug {
				strainPostSlider := 0.0
				if strainBeforeSlider != strain && oldstrainBeforeDecay == 0 {
					strainPostSlider = strain
				} else {
					strainPostSlider = strainBeforeSlider
				}

				distRaw := md.AimPoints[i].Pos.DistanceFrom(md.AimPoints[i-1].Pos)
				debugf("[STRAIN] i=%v distRaw=%v distW=%v interval=%v timeW=%v angleBonus=%v strainPreSlider=%v strainPostSlider=%v oldstrainBeforeDecay=%v oldstrainAfterDecay=%v finalStrain=%v\n",
					i,
					distRaw,
					distance,
					interval,
					time,
					angleBonus,
					strainBeforeSlider,
					strainPostSlider,
					oldstrainBeforeDecay,
					oldStrain,
					strain,
				)
			}
		}
		md.AimStrains = append(md.AimStrains, strain)
		oldStrain = strain
	}
}
