package skills

import "github.com/compico/go-osu/pkg/osu"

func ApplyModsToBeatmap(original *osu.Beatmap, mods osu.Mod) *osu.Beatmap {
	// Создаём копию
	bm := *original

	if mods&osu.HD != 0 || mods&osu.FL != 0 {
		// HD/FL не меняют AR/OD/CS
	}
	if mods&osu.HR != 0 {
		bm.CircleSize = min(10, bm.CircleSize*1.3)
		bm.OverallDifficulty = min(10, bm.OverallDifficulty*1.4)
		bm.ApproachRate = min(10, bm.ApproachRate*1.4)
		bm.HPDrainRate = min(10, bm.HPDrainRate*1.4)
	}
	if mods&osu.EZ != 0 {
		bm.CircleSize = max(0, bm.CircleSize/2)
		bm.OverallDifficulty = max(0, bm.OverallDifficulty/2)
		bm.ApproachRate = max(0, bm.ApproachRate/2)
		bm.HPDrainRate = max(0, bm.HPDrainRate/2)
	}
	// DT/HT меняют ТАЙМИНГ, а не AR напрямую — их обрабатывают отдельно при расчёте preempt
	return &bm
}
