package response

import (
	"github.com/compico/go-osu/pkg/osu/skills"
)

// OsuStrainsResponse is the payload of GET /api/osu/songs/{id}/strains.
//
// Every curve is a list of {time, value} points ordered by time. Time is in
// milliseconds on the timeline the skills are calculated on: the beatmap's own
// time divided by Speed (so with DT, a point at the audio's 10 s mark of the
// original has time 6667). Multiply by Speed to get the beatmap's own time.
// Value is on the scale of the skill, so a curve's maximum equals the skill
// shown on the beatmap card (Memory is cumulative and ends at the skill,
// Tenacity is a step curve; see skills.Curves).
//
// Curves are thinned to the strongest point per Window milliseconds and are
// sparse where the map has breaks. A curve that does not exist for the map
// (Memory without FL, Tenacity without streams) is an empty list.
type OsuStrainsResponse struct {
	BeatmapID int32   `json:"beatmap_id"`
	Mods      string  `json:"mods"`
	Speed     float64 `json:"speed"`
	Window    int     `json:"window"`

	Stamina   []skills.StrainPoint `json:"stamina"`
	Tenacity  []skills.StrainPoint `json:"tenacity"`
	Agility   []skills.StrainPoint `json:"agility"`
	Precision []skills.StrainPoint `json:"precision"`
	Reading   []skills.StrainPoint `json:"reading"`
	Memory    []skills.StrainPoint `json:"memory"`
	Accuracy  []skills.StrainPoint `json:"accuracy"`
	Reaction  []skills.StrainPoint `json:"reaction"`
}

// NewOsuStrainsResponse thins every curve to the strongest point per window
// (0 keeps them all).
func NewOsuStrainsResponse(beatmapID int32, curves *skills.Curves, window int) OsuStrainsResponse {
	thin := func(points []skills.StrainPoint) []skills.StrainPoint {
		out := skills.MaxPerWindow(points, window)
		if out == nil {
			// Encode "no curve" as [] and not as null.
			out = []skills.StrainPoint{}
		}
		return out
	}

	return OsuStrainsResponse{
		BeatmapID: beatmapID,
		Mods:      curves.ModsString,
		Speed:     curves.Speed,
		Window:    window,

		Stamina:   thin(curves.Stamina),
		Tenacity:  thin(curves.Tenacity),
		Agility:   thin(curves.Agility),
		Precision: thin(curves.Precision),
		Reading:   thin(curves.Reading),
		Memory:    thin(curves.Memory),
		Accuracy:  thin(curves.Accuracy),
		Reaction:  thin(curves.Reaction),
	}
}
