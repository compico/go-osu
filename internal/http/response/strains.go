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
// Window is in milliseconds of the beatmap's own timeline. Unless it is 0,
// every curve is resampled onto the same uniform grid of such windows (the
// middle of window k is at (k+0.5)*Window of the beatmap's time): one point per
// window holding the strongest value in it (see skills.ResampleMax), 0 where the
// map has a break or no streams, and for memory, which only adds up, the value so
// far. With Window 0 every point of every curve is returned as it is. A curve
// that does not exist for the map (Memory without FL) is an empty list.
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

// NewOsuStrainsResponse resamples every curve onto windows of `window` ms of
// the beatmap's timeline (0 keeps every point as it is).
func NewOsuStrainsResponse(beatmapID int32, curves *skills.Curves, window int) OsuStrainsResponse {
	windows := skills.WindowCount(window, curves.Speed,
		curves.Stamina, curves.Tenacity, curves.Agility, curves.Precision,
		curves.Reading, curves.Memory, curves.Accuracy, curves.Reaction)

	resample := func(points []skills.StrainPoint, carry bool) []skills.StrainPoint {
		// A curve the map does not have stays empty, not a list of zeros.
		if len(points) == 0 {
			return []skills.StrainPoint{}
		}
		return skills.ResampleMax(points, window, curves.Speed, windows, carry)
	}

	return OsuStrainsResponse{
		BeatmapID: beatmapID,
		Mods:      curves.ModsString,
		Speed:     curves.Speed,
		Window:    window,

		Stamina:   resample(curves.Stamina, false),
		Tenacity:  resample(curves.Tenacity, false),
		Agility:   resample(curves.Agility, false),
		Precision: resample(curves.Precision, false),
		Reading:   resample(curves.Reading, false),
		Memory:    resample(curves.Memory, true),
		Accuracy:  resample(curves.Accuracy, false),
		Reaction:  resample(curves.Reaction, false),
	}
}
