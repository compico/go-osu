package skills

import (
	"testing"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

func TestPrepareMapData_FollowsOriginalPipeline(t *testing.T) {
	bm := &osu.Beatmap{
		ApproachRate: 9,
		HitObjects: []osu.HitObject{
			{
				Time: 100,
				Pos:  vector2d.Vector2dd{X: 100, Y: 100},
				Type: int(osu.HitNormal),
			},
			{
				Time:        400,
				Pos:         vector2d.Vector2dd{X: 200, Y: 100},
				Type:        int(osu.HitSlider),
				PixelLength: 1,
				Repeat:      1,
			},
			{
				Time: 700,
				Pos:  vector2d.Vector2dd{X: 300, Y: 100},
				Type: int(osu.HitNormal),
			},
		},
	}

	md := NewMapData(bm, 0)
	PrepareMapData(md)

	wantIntervals := []int{300, 300}
	if len(md.PressIntervals) != len(wantIntervals) {
		t.Fatalf("expected %d press intervals, got %d", len(wantIntervals), len(md.PressIntervals))
	}
	for i, want := range wantIntervals {
		if md.PressIntervals[i] != want {
			t.Fatalf("press interval %d: got %d, want %d", i, md.PressIntervals[i], want)
		}
	}

	if len(md.AimPoints) != 3 {
		t.Fatalf("expected 3 aim points for 3 hit objects, got %d", len(md.AimPoints))
	}
}
