package skills

import "github.com/compico/go-osu/pkg/osu"

func ProcessBeatmap(bm *osu.Beatmap, mods osu.Mod, vars *Vars) *SkillResult {
	if debug {
		bmBpm := bm.BPM()
		debugf(
			"Original Stats: ar=%v od=%v cs=%v hp=%v bpm=%.2f-%.2f(%.2f)\n",
			bm.ApproachRate,
			bm.OverallDifficulty,
			bm.CircleSize,
			bm.HPDrainRate,
			bmBpm.Min,
			bmBpm.Max,
			bmBpm.Avg,
		)
	}

	modifiedBm := ApplyMods(bm, mods)
	if debug {
		modifiedBmBpm := bm.BPM()
		debugf(
			"After ApplyMods Stats: ar=%v od=%v cs=%v hp=%v bpm=%.2f-%.2f(%.2f)\n",
			modifiedBm.ApproachRate,
			modifiedBm.OverallDifficulty,
			modifiedBm.CircleSize,
			modifiedBm.HPDrainRate,
			modifiedBmBpm.Min,
			modifiedBmBpm.Max,
			modifiedBmBpm.Avg,
		)
	}

	md := NewMapData(modifiedBm, mods)
	if debug {
		mdBpm := md.Map.BPM()
		debugf(
			"Create MapData Stats: ar=%v od=%v cs=%v hp=%v bpm=%.2f-%.2f(%.2f)\n",
			md.Map.ApproachRate,
			md.Map.OverallDifficulty,
			md.Map.CircleSize,
			md.Map.HPDrainRate,
			mdBpm.Min,
			mdBpm.Max,
			mdBpm.Avg,
		)
	}

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
