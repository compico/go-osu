package skills

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/beatmap"
)

// TestSkillsValues_OutputForComparison outputs calculated skill values for all
// beatmaps in test_data with no mods, to compare against the original C++ implementation.
func TestSkillsValues_OutputForComparison(t *testing.T) {
	files, err := getOsuFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Skip("no .osu files found in ./test_data")
	}

	vars := DefaultVars()

	fmt.Println("\n=== Skill Calculation Results (No Mods) ===")
	fmt.Println("Format: beatmap_id | stamina | tenacity | agility | precision | reading | memory | accuracy | reaction")
	fmt.Println("---")

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Errorf("error reading %s: %v", f, err)
			continue
		}

		var bm osu.Beatmap
		if err := beatmap.Unmarshal(data, &bm); err != nil {
			t.Errorf("parse error in %s: %v", f, err)
			continue
		}

		if bm.Mode != osu.ModeOsu {
			continue
		}

		// Calculate with no mods
		result := ProcessBeatmap(&bm, 0, vars)

		// Format: beatmap_id | stamina | tenacity | agility | precision | reading | memory | accuracy | reaction
		debugf("%d | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f\n",
			bm.BeatmapID,
			result.Skills.Stamina,
			result.Skills.Tenacity,
			result.Skills.Agility,
			result.Skills.Precision,
			result.Skills.Reading,
			result.Skills.Memory,
			result.Skills.Accuracy,
			result.Skills.Reaction,
		)

		// Also test with common mod combinations
		modCombinations := []struct {
			name string
			mods osu.Mod
		}{
			{"DT", osu.DT},
			{"HR", osu.HR},
			{"HDDT", osu.HD | osu.DT},
		}

		for _, modCombo := range modCombinations {
			result := ProcessBeatmap(&bm, modCombo.mods, vars)

			debugf("%d (%s) | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f\n",
				bm.BeatmapID,
				modCombo.name,
				result.Skills.Stamina,
				result.Skills.Tenacity,
				result.Skills.Agility,
				result.Skills.Precision,
				result.Skills.Reading,
				result.Skills.Memory,
				result.Skills.Accuracy,
				result.Skills.Reaction,
			)
		}

		fmt.Println("---")
	}

	t.Log("Skill values output complete — see stdout for results")
}

func TestSkillsValues_Race(t *testing.T) {
	files, err := getOsuFiles()
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Skip("no .osu files found in ./test_data")
	}

	vars := DefaultVars()

	for _, f := range files {
		f := f // фиксируем переменную цикла

		t.Run(filepath.Base(f), func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("error reading %s: %v", f, err)
			}

			var bm osu.Beatmap
			if err := beatmap.Unmarshal(data, &bm); err != nil {
				t.Fatalf("parse error in %s: %v", f, err)
			}

			if bm.Mode != osu.ModeOsu {
				t.Skip("not osu mode")
			}

			// Несколько одновременных расчётов одного beatmap
			var wg sync.WaitGroup

			for i := 0; i < 100; i++ {
				wg.Add(1)

				go func() {
					defer wg.Done()

					result := ProcessBeatmap(&bm, 0, vars)

					_ = result.Skills.Stamina
					_ = result.Skills.Agility
					_ = result.Skills.Precision
				}()
			}

			wg.Wait()
		})
	}
}

type SkillsReference struct {
	BeatmapID int
	Mods      osu.Mod
	Skills    Skills
}

func TestSkillsValues_CompareWithReference(t *testing.T) {
	files, err := getOsuFiles()
	if err != nil {
		t.Fatal(err)
	}

	vars := DefaultVars()

	const epsilon = 0.001

	check := func(t *testing.T, name string, got, want float64) {
		t.Helper()

		diff := math.Abs(got - want)

		if diff > epsilon {
			t.Errorf("%s FAIL: got %.6f want %.6f (diff %.6f)",
				name, got, want, diff)
			return
		}

		t.Logf("%s OK: %.6f (expected %.6f)",
			name, got, want)
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Errorf("error reading %s: %v", f, err)
			continue
		}

		var bm osu.Beatmap
		if err := beatmap.Unmarshal(data, &bm); err != nil {
			t.Errorf("parse error in %s: %v", f, err)
			continue
		}

		for _, ref := range skillsReference {
			if ref.BeatmapID != bm.BeatmapID {
				continue
			}

			t.Run(fmt.Sprintf("%d_mods_%v", ref.BeatmapID, ref.Mods.String()), func(t *testing.T) {
				result := ProcessBeatmap(&bm, ref.Mods, vars)

				check(t, "Stamina",
					result.Skills.Stamina,
					ref.Skills.Stamina)

				check(t, "Tenacity",
					result.Skills.Tenacity,
					ref.Skills.Tenacity)

				check(t, "Agility",
					result.Skills.Agility,
					ref.Skills.Agility)

				check(t, "Precision",
					result.Skills.Precision,
					ref.Skills.Precision)

				check(t, "Reading",
					result.Skills.Reading,
					ref.Skills.Reading)

				check(t, "Memory",
					result.Skills.Memory,
					ref.Skills.Memory)

				check(t, "Accuracy",
					result.Skills.Accuracy,
					ref.Skills.Accuracy)

				check(t, "Reaction",
					result.Skills.Reaction,
					ref.Skills.Reaction)
			})
		}
	}
}
