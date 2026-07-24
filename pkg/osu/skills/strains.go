package skills

import "math"

// strains.cpp. It turns the press intervals into the per-note strain curve
// that stamina uses for the final score.
func calculateTapStrains(md *MapData, vars *Vars) {
	count := 0
	strain := 0.0
	oldBonus := 0.0

	for _, interval := range md.PressIntervals {
		intervalF := float64(interval)

		if count == 0 {
			if intervalF >= vars.Get("Stamina", "LargestInterval") {
				strain = 0
			} else {
				strain = vars.Get("Stamina", "Scale") /
					math.Pow(
						intervalF,
						math.Pow(intervalF, vars.Get("Stamina", "Pow"))*vars.Get("Stamina", "Mult"),
					)
			}

			md.TapStrains = append(md.TapStrains, strain)
		} else {
			if intervalF >= vars.Get("Stamina", "LargestInterval") {
				strain *= vars.Get("Stamina", "DecayMax")
			} else {
				if interval <= 15 {
					continue
				}

				strain = vars.Get("Stamina", "Scale") /
					math.Pow(
						intervalF,
						math.Pow(intervalF, vars.Get("Stamina", "Pow"))*vars.Get("Stamina", "Mult"),
					)

				strain += oldBonus * vars.Get("Stamina", "Decay")
			}

			md.TapStrains = append(md.TapStrains, strain)
		}

		oldBonus = strain
		count++
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

// calculateAimStrains вычисляет страйны для Agility v2.
// Требует md.AimPoints, md.AngleBonuses (заполняются в calculateAnglesAndBonuses).
func calculateAimStrains(md *MapData, vars *Vars) {
	md.AimStrains = make([]float64, 0, len(md.AimPoints))
	oldStrain := 0.0

	// debug vars
	strainBeforeSlider := 0.0
	oldstrainBeforeDecay := 0.0

	for i := 0; i < len(md.AimPoints); i++ {
		strain := 0.0
		if i > 0 {
			distance := getWeightedAimDistance(md.AimPoints[i].Pos.DistanceFrom(md.AimPoints[i-1].Pos), vars)
			interval := md.AimPoints[i].Time - md.AimPoints[i-1].Time
			time := getWeightedAimTime(float64(interval), vars)

			angleBonus := 1.0
			if i > 1 {
				angleBonus = 1 + (vars.Get("Agility", "AngleMult") * md.AngleBonuses[i-2])
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
				strain *= vars.Get("Agility", "SliderStrainDecay")
			}

			if debug {
				oldstrainBeforeDecay = oldStrain
			}
			oldStrain -= vars.Get("Agility", "StrainDecay") * float64(interval)
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
