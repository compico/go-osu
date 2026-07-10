package repository

import "github.com/compico/go-osu/internal/database"

type Repos struct {
	Settings    *SettingsRepo
	BeatmapSets *BeatmapSetRepo
	Beatmaps    *BeatmapRepo
	Skills      *SkillRepo
}

func New(db *database.DB) *Repos {
	return &Repos{
		Settings:    NewSettingsRepo(db),
		BeatmapSets: NewBeatmapSetRepo(db),
		Beatmaps:    NewBeatmapRepo(db),
		Skills:      NewSkillRepo(db),
	}
}
