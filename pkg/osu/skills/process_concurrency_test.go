package skills

import (
	"math"
	"reflect"
	"sync"
	"testing"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

func TestProcessBeatmap_ConcurrentSameBeatmapWithDifferentMods(t *testing.T) {
	beatmap := concurrentTestBeatmap()
	vars := NewVars()
	mods := []osu.Mod{
		osu.NM,
		osu.HD,
		osu.HR,
		osu.EZ,
		osu.DT,
		osu.HT,
		osu.FL,
		osu.HR | osu.HD,
		osu.HD | osu.DT,
		osu.EZ | osu.HT,
	}

	expected := make(map[osu.Mod]SkillResult, len(mods))
	for _, mod := range mods {
		expected[mod] = *ProcessBeatmap(beatmap, mod, vars)
	}

	before := *beatmap
	before.TimingPoints = append([]osu.TimingPoint(nil), beatmap.TimingPoints...)
	before.HitObjects = append([]osu.HitObject(nil), beatmap.HitObjects...)

	type calculation struct {
		mod    osu.Mod
		result *SkillResult
	}

	const runsPerMod = 8
	results := make(chan calculation, len(mods)*runsPerMod)
	var wg sync.WaitGroup
	for run := 0; run < runsPerMod; run++ {
		for _, mod := range mods {
			mod := mod
			wg.Add(1)
			go func() {
				defer wg.Done()
				results <- calculation{
					mod:    mod,
					result: ProcessBeatmap(beatmap, mod, vars),
				}
			}()
		}
	}

	wg.Wait()
	close(results)

	for calculation := range results {
		want := expected[calculation.mod]
		if calculation.result == nil || !skillsAreFinite(calculation.result.Skills) {
			t.Fatalf("calculation for %s returned nil or non-finite skills: %#v", calculation.mod, calculation.result)
		}
		if *calculation.result != want {
			t.Errorf("calculation for %s changed between sequential and concurrent runs:\n got: %#v\nwant: %#v",
				calculation.mod, *calculation.result, want)
		}
	}

	if !reflect.DeepEqual(before, *beatmap) {
		t.Fatal("ProcessBeatmap modified the source beatmap")
	}
}

func concurrentTestBeatmap() *osu.Beatmap {
	beatmap := &osu.Beatmap{
		ApproachRate:      8,
		OverallDifficulty: 8,
		CircleSize:        4,
		HPDrainRate:       5,
		TimingPoints: []osu.TimingPoint{{
			Time:        0,
			BeatLength:  500,
			Meter:       4,
			Uninherited: true,
		}},
		HitObjects: make([]osu.HitObject, 24),
	}

	for i := range beatmap.HitObjects {
		beatmap.HitObjects[i] = osu.HitObject{
			Pos: vector2d.Vector2dd{
				X: float64((i*71)%590 + 20),
				Y: float64((i*i*43+i*29)%430 + 20),
			},
			Time: i*350 + 100,
			Type: osu.HitNormal,
		}
	}

	return beatmap
}

func skillsAreFinite(skills Skills) bool {
	for _, value := range []float64{
		skills.Stamina,
		skills.Tenacity,
		skills.Agility,
		skills.Precision,
		skills.Reading,
		skills.Memory,
		skills.Accuracy,
		skills.Reaction,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}
