package osuapi

import (
	"github.com/compico/go-osu/internal/model"
	"github.com/compico/go-osu/internal/repository"
)

type GroupDTO struct {
	BeatmapSetID int32     `json:"beatmap_set_id"`
	SongTitle    string    `json:"song_title"`
	ArtistName   string    `json:"artist_name"`
	Diffs        []DiffDTO `json:"diffs"`
	StarsMin     float64   `json:"stars_min"`
	StarsMax     float64   `json:"stars_max"`
}

type DiffDTO struct {
	BeatmapID    int32      `json:"beatmap_id"`
	Difficulty   string     `json:"difficulty"`
	Stars        float64    `json:"stars"`
	ApproachRate float32    `json:"approach_rate"`
	DrainTime    int32      `json:"drain_time"`
	BPM          float64    `json:"bpm"`
	AudioFile    string     `json:"audio_file_name"`
	CreatorName  string     `json:"creator_name"`
	Skills       *SkillsDTO `json:"skills,omitempty"` // nil when no mods= given
}

type SkillsDTO struct {
	Stamina   float64 `json:"stamina"`
	Tenacity  float64 `json:"tenacity"`
	Agility   float64 `json:"agility"`
	Precision float64 `json:"precision"`
	Reading   float64 `json:"reading"`
	Memory    float64 `json:"memory"`
	Accuracy  float64 `json:"accuracy"`
	Reaction  float64 `json:"reaction"`
}

func toGroupDTO(g repository.GroupResult, skills map[int32]model.SkillCache) GroupDTO {
	diffs := make([]DiffDTO, 0, len(g.Diffs))
	starsMin, starsMax := 0.0, 0.0
	for i, d := range g.Diffs {
		if i == 0 || d.StarsNoMod < starsMin {
			starsMin = d.StarsNoMod
		}
		if d.StarsNoMod > starsMax {
			starsMax = d.StarsNoMod
		}

		dto := DiffDTO{
			BeatmapID:    d.BeatmapID,
			Difficulty:   d.Difficulty,
			Stars:        d.StarsNoMod,
			ApproachRate: d.ApproachRate,
			DrainTime:    d.DrainTime,
			BPM:          d.BPM,
			AudioFile:    d.AudioFileName,
			CreatorName:  g.Set.CreatorName,
		}
		if sk, ok := skills[d.BeatmapID]; ok {
			dto.Skills = &SkillsDTO{
				Stamina: sk.Stamina, Tenacity: sk.Tenacity, Agility: sk.Agility, Precision: sk.Precision,
				Reading: sk.Reading, Memory: sk.Memory, Accuracy: sk.Accuracy, Reaction: sk.Reaction,
			}
		}
		diffs = append(diffs, dto)
	}

	return GroupDTO{
		BeatmapSetID: g.Set.BeatmapSetID,
		SongTitle:    g.Set.SongTitle,
		ArtistName:   g.Set.ArtistName,
		Diffs:        diffs,
		StarsMin:     starsMin,
		StarsMax:     starsMax,
	}
}
