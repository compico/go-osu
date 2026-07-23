package skills

import "github.com/compico/go-osu/pkg/osu"

func ProcessBeatmap(bm *osu.Beatmap, mods osu.Mod, vars *Vars) *SkillResult {
	modifiedBm := ApplyMods(bm, mods)
	md := NewMapData(modifiedBm, mods)

	prepareMapData(md)

	calculateAimStrains(md, vars)
	calculateTapStrains(md, vars)
	calculateSkills(md, vars)

	return &SkillResult{
		Mods:       mods,
		ModsString: mods.String(),
		Skills:     md.Skills,
	}
}
