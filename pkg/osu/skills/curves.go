package skills

import "github.com/compico/go-osu/pkg/osu"

// StrainPoint is one point of a skill's difficulty curve.
type StrainPoint struct {
	// Time in milliseconds on the timeline the skills are calculated on: the
	// beatmap's own time (the same as hit object times in the .osu file),
	// divided by Curves.Speed when DT or HT is on.
	Time int `json:"time"`
	// Value on the scale of the skill itself, so the curve is comparable with
	// the skill's value in Skills and with other skills' curves. For skills
	// that take their value from the strongest strain, the curve's maximum is
	// that value.
	Value float64 `json:"value"`
}

// Curves holds how each skill's difficulty develops over the course of a
// beatmap. Every curve is ordered by time and has one point per note the skill
// is measured on, without smoothing or downsampling.
type Curves struct {
	Mods       osu.Mod
	ModsString string

	// Speed is the playback speed the mods imply (1.5 for DT, 0.75 for HT,
	// otherwise 1). Multiply a point's Time by it to get the time in the
	// beatmap's own timeline.
	Speed float64

	Stamina []StrainPoint
	Agility []StrainPoint
	Reading []StrainPoint
}

// ProcessCurves calculates the difficulty curves of a beatmap for the given
// mods. It is independent of ProcessBeatmap: the skills' values are not
// calculated here and calculating curves does not change them. Like
// ProcessBeatmap it does not modify bm and can run concurrently on the same
// beatmap.
func ProcessCurves(bm *osu.Beatmap, mods osu.Mod, vars *Vars) *Curves {
	curves := &Curves{
		Mods:       mods,
		ModsString: mods.String(),
		Speed:      modsSpeed(mods),
	}

	modifiedBm := ApplyMods(bm, mods)
	if len(modifiedBm.HitObjects) == 0 {
		return curves
	}

	md := NewMapData(modifiedBm, mods)
	md.prepare()

	md.calculateTapStrains(vars)
	curves.Stamina = md.staminaCurve(vars)

	md.calculateAimStrains(vars)
	curves.Agility = md.agilityCurve(vars)

	md.calculateReading(vars, md.HasMod(osu.HD))
	curves.Reading = md.readingCurve(vars)

	return curves
}

// modsSpeed returns the playback speed multiplier of the speed-changing mods.
func modsSpeed(mods osu.Mod) float64 {
	switch {
	case mods&osu.DT != 0:
		return 1.5
	case mods&osu.HT != 0:
		return 0.75
	default:
		return 1
	}
}

// staminaCurve returns the tap strain of every press on the stamina scale.
// Requires calculateTapStrains.
func (md *MapData) staminaCurve(vars *Vars) []StrainPoint {
	points := make([]StrainPoint, len(md.TapStrains))
	for i, strain := range md.TapStrains {
		points[i] = StrainPoint{
			Time:  md.TapStrainTimes[i],
			Value: staminaScale(strain, vars),
		}
	}
	return points
}

// agilityCurve returns the aim strain of every aim point on the agility scale.
// Requires calculateAimStrains.
//
// The agility skill is not the strongest strain but a weighted sum of the
// strain peaks, so applying the skill's formula to a single strain would put
// the curve below the skill. Instead the strains are scaled by the factor that
// turns the strongest of them into the skill's value. The shape is kept
// (a value is proportional to its strain raised to Agility.TotalPow), and the
// curve's maximum is exactly Skills.Agility.
//
// A map without strain peaks (a few notes, or strains that only rise) has an
// agility of 0, and so has its curve.
func (md *MapData) agilityCurve(vars *Vars) []StrainPoint {
	maxStrain := MaxFloatSliceValue(md.AimStrains)
	weighted := getWeightedValue2(getPeakVals(md.AimStrains), vars.Agility.Weighting)

	points := make([]StrainPoint, len(md.AimStrains))
	for i, strain := range md.AimStrains {
		// For the strongest strain the ratio is exactly 1, which makes the
		// maximum of the curve exactly the skill. All strains are 0 when
		// maxStrain is, and so is the skill.
		aggregated := 0.0
		if maxStrain > 0 {
			aggregated = weighted * (strain / maxStrain)
		}

		points[i] = StrainPoint{
			Time:  md.AimStrainTimes[i],
			Value: agilityScale(aggregated, vars),
		}
	}
	return points
}

// readingCurve returns the reading strain of every reading point on the
// reading scale. Requires calculateReading.
//
// Reading is a weighted sum of strain peaks, like agility, so the curve is
// built the same way: the strains are scaled by the factor that turns the
// strongest of them into the skill's value, keeping the shape (a value is
// proportional to its strain raised to Reading.TotalPow). The curve's maximum
// is exactly Skills.Reading.
//
// There is a strain for every reading point, so ReadingPoints gives the times.
func (md *MapData) readingCurve(vars *Vars) []StrainPoint {
	maxStrain := MaxFloatSliceValue(md.ReadingStrains)
	weighted := getWeightedValue2(getPeakVals(md.ReadingStrains), vars.Reading.Weighting)

	points := make([]StrainPoint, len(md.ReadingStrains))
	for i, strain := range md.ReadingStrains {
		aggregated := 0.0
		if maxStrain > 0 {
			aggregated = weighted * (strain / maxStrain)
		}

		points[i] = StrainPoint{
			Time:  md.ReadingPoints[i].Time,
			Value: readingScale(aggregated, vars),
		}
	}
	return points
}
