package skills

import (
	"math"
	"testing"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

func TestApplyMods_HR(t *testing.T) {
	bm := &osu.Beatmap{
		ApproachRate:      5,
		OverallDifficulty: 5,
		HPDrainRate:       5,
		CircleSize:        4,
		HitObjects: []osu.HitObject{
			{Pos: vector2d.Vector2dd{X: 100, Y: 200}},
		},
	}

	result := ApplyMods(bm, osu.HR)

	// AR: 5 * 1.4 = 7
	if result.ApproachRate != 7 {
		t.Errorf("HR AR: got %f, want 7", result.ApproachRate)
	}
	// CS: 4 * 1.3 = 5.2
	if result.CircleSize != 5.2 {
		t.Errorf("HR CS: got %f, want 5.2", result.CircleSize)
	}
	// Flip Y: 384 - 200 = 184
	if result.HitObjects[0].Pos.Y != 184 {
		t.Errorf("HR flip Y: got %f, want 184", result.HitObjects[0].Pos.Y)
	}
	// Оригинал не изменён
	if bm.HitObjects[0].Pos.Y != 200 {
		t.Errorf("original modified: Y=%f", bm.HitObjects[0].Pos.Y)
	}
}

func TestApplyMods_EZ(t *testing.T) {
	bm := &osu.Beatmap{
		ApproachRate:      8,
		OverallDifficulty: 8,
		HPDrainRate:       8,
		CircleSize:        6,
	}

	result := ApplyMods(bm, osu.EZ)

	if result.ApproachRate != 4 {
		t.Errorf("EZ AR: got %f, want 4", result.ApproachRate)
	}
	if result.CircleSize != 3 {
		t.Errorf("EZ CS: got %f, want 3", result.CircleSize)
	}
}

func TestApplyMods_DT(t *testing.T) {
	bm := &osu.Beatmap{
		ApproachRate:      5,
		OverallDifficulty: 5,
		HPDrainRate:       5,
		HitObjects: []osu.HitObject{
			{Time: 1500, EndTime: 2000},
		},
		TimingPoints: []osu.TimingPoint{
			{Time: 0, BeatLength: 500, Uninherited: true},
		},
	}

	result := ApplyMods(bm, osu.DT)

	// AR: 5 * 1.5 = 7.5
	if result.ApproachRate != 7.5 {
		t.Errorf("DT AR: got %f, want 7.5", result.ApproachRate)
	}
	// Time: 1500 / 1.5 = 1000
	if result.HitObjects[0].Time != 1000 {
		t.Errorf("DT time: got %d, want 1000", result.HitObjects[0].Time)
	}
	// BeatLength: 500 / 1.5
	if math.Abs(result.TimingPoints[0].BeatLength-500.0/1.5) > 0.001 {
		t.Errorf("DT beatLength: got %f, want %f",
			result.TimingPoints[0].BeatLength, 500.0/1.5)
	}
}

func TestApplyMods_HT(t *testing.T) {
	bm := &osu.Beatmap{
		ApproachRate:      5,
		OverallDifficulty: 5,
		HPDrainRate:       5,
		HitObjects: []osu.HitObject{
			{Time: 1500, EndTime: 2000},
		},
		TimingPoints: []osu.TimingPoint{
			{Time: 0, BeatLength: 500, Uninherited: true},
		},
	}

	result := ApplyMods(bm, osu.HT)

	// AR: 5 * 0.75 = 3.75
	if result.ApproachRate != 3.75 {
		t.Errorf("HT AR: got %f, want 3.75", result.ApproachRate)
	}
	// Time: 1500 / 0.75 = 2000
	if result.HitObjects[0].Time != 2000 {
		t.Errorf("HT time: got %d, want 2000", result.HitObjects[0].Time)
	}
}

func TestApplyMods_NoMods(t *testing.T) {
	bm := &osu.Beatmap{ApproachRate: 9}
	result := ApplyMods(bm, 0)
	if result != bm {
		t.Error("no mods should return original pointer")
	}
}

func TestApplyMods_HighAR(t *testing.T) {
	bm := &osu.Beatmap{ApproachRate: 10}

	// DT: (10-5)*1.5 + 5 = 12.5 → clamp to 10
	result := ApplyMods(bm, osu.DT)
	if result.ApproachRate != 10 {
		t.Errorf("DT AR(10): got %f, want 10", result.ApproachRate)
	}

	// HT: (10-5)*0.75 + 5 = 8.75
	result = ApplyMods(bm, osu.HT)
	if math.Abs(result.ApproachRate-8.75) > 0.001 {
		t.Errorf("HT AR(10): got %f, want 8.75", result.ApproachRate)
	}
}

func TestApplyMods_HR_CurvesFlipped(t *testing.T) {
	bm := &osu.Beatmap{
		HitObjects: []osu.HitObject{
			{
				Pos:    vector2d.Vector2dd{X: 100, Y: 100},
				Curves: []vector2d.Vector2dd{{X: 200, Y: 200}, {X: 300, Y: 50}},
			},
		},
	}

	result := ApplyMods(bm, osu.HR)

	if result.HitObjects[0].Pos.Y != 284 {
		t.Errorf("HR Pos.Y: got %f, want 284", result.HitObjects[0].Pos.Y)
	}
	if result.HitObjects[0].Curves[0].Y != 184 {
		t.Errorf("HR Curves[0].Y: got %f, want 184", result.HitObjects[0].Curves[0].Y)
	}
	if result.HitObjects[0].Curves[1].Y != 334 {
		t.Errorf("HR Curves[1].Y: got %f, want 334", result.HitObjects[0].Curves[1].Y)
	}
}
