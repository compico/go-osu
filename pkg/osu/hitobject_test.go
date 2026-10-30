package osu

import (
	"testing"

	"github.com/compico/go-osu/pkg/vector2d"
)

func TestHitObjectCloneCopiesSlices(t *testing.T) {
	source := HitObject{
		Curves: []vector2d.Vector2dd{{X: 1, Y: 2}},
		LerpPoints: []vector2d.Vector2dd{
			{X: 3, Y: 4},
		},
		RepeatTimes: []int{5},
		Ticks:       []int{6},
	}
	clone := source.Clone()

	clone.Curves[0].X = 10
	clone.LerpPoints[0].X = 30
	clone.RepeatTimes[0] = 50
	clone.Ticks[0] = 60

	if source.Curves[0].X != 1 || source.LerpPoints[0].X != 3 || source.RepeatTimes[0] != 5 || source.Ticks[0] != 6 {
		t.Fatal("mutating cloned slices changed the source hit object")
	}
}
