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
