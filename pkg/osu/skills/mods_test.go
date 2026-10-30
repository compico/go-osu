package skills

import (
	"testing"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

func TestApplyModsCopiesHitObjectData(t *testing.T) {
	original := &osu.Beatmap{
		HitObjects: []osu.HitObject{{
			Type:        osu.HitSlider,
			Curves:      []vector2d.Vector2dd{{X: 1, Y: 2}},
			LerpPoints:  []vector2d.Vector2dd{{X: 3, Y: 4}},
			RepeatTimes: []int{5},
			Ticks:       []int{6},
		}},
	}

	modified := ApplyMods(original, osu.NM)
	modified.HitObjects[0].Curves[0].X = 10
	modified.HitObjects[0].LerpPoints[0].X = 30
	modified.HitObjects[0].RepeatTimes[0] = 50
	modified.HitObjects[0].Ticks[0] = 60

	if got := original.HitObjects[0]; got.Curves[0].X != 1 || got.LerpPoints[0].X != 3 || got.RepeatTimes[0] != 5 || got.Ticks[0] != 6 {
		t.Fatal("modded hit object shares nested slices with the source beatmap")
	}
}
