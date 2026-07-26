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

var skillsReference = []SkillsReference{
	{
		BeatmapID: 215,
		Mods:      osu.NM,
		Skills: Skills{
			Stamina:   153.68613250423354,
			Tenacity:  260.60459914572215,
			Agility:   328.0786343363215,
			Accuracy:  329.47524173163384,
			Precision: 311.98296054385639,
			Reaction:  217.29977149243206,
			Reading:   0,
			Memory:    0,
		},
	},
	{
		BeatmapID: 215,
		Mods:      osu.HD | osu.HR | osu.FL,
		Skills: Skills{
			Stamina:   153.68613250423354,
			Tenacity:  260.60459914572215,
			Agility:   328.0786343363215,
			Accuracy:  614.58436899159585,
			Precision: 527.25120331911728,
			Reaction:  448.95191431411928,
			Reading:   1.8700417757736414,
			Memory:    337.20227537901644,
		},
	},
	{
		BeatmapID: 5066071,
		Mods:      osu.NM,
		Skills: Skills{
			Stamina:   1001.1464705249988,
			Tenacity:  564.02768764656082,
			Agility:   800.75799745057759,
			Accuracy:  1275.6548780816715,
			Precision: 494.46212879800419,
			Reaction:  424.54960111644277,
			Reading:   2.6594654715874784,
			Memory:    0,
		},
	},
	{
		BeatmapID: 2156120,
		Mods:      osu.NM,
		Skills: Skills{
			Stamina:   140.86958811646369,
			Tenacity:  153.39704219353868,
			Agility:   491.43427229223346,
			Accuracy:  240.72240402598004,
			Precision: 238.51791673702124,
			Reaction:  216.76747026796528,
			Reading:   1.268459022080946,
			Memory:    0,
		},
	},
	{
		BeatmapID: 2156120,
		Mods:      osu.DT | osu.HD,
		Skills: Skills{
			Stamina:   252.66232425750019,
			Tenacity:  231.78734310466587,
			Agility:   1096.502734268279,
			Accuracy:  512.13824999656049,
			Precision: 331.91714464460085,
			Reaction:  385.27193087390447,
			Reading:   7.9436465296778938,
			Memory:    0,
		},
	},
	{
		BeatmapID: 3398802,
		Mods:      osu.NM,
		Skills: Skills{
			Stamina:   227.99391325634804,
			Tenacity:  220.27730795451728,
			Agility:   570.10769571937385,
			Accuracy:  397.24740119334484,
			Precision: 229.3259069817596,
			Reaction:  303.36730582762209,
			Reading:   2.4237296096243619,
			Memory:    0,
		},
	},
	{
		BeatmapID: 3398802,
		Mods:      osu.HR | osu.HD,
		Skills: Skills{
			Stamina:   227.99391325634804,
			Tenacity:  220.27730795451728,
			Agility:   570.10769662603536,
			Accuracy:  668.46235652176449,
			Precision: 387.56078305969339,
			Reaction:  449.13499486768518,
			Reading:   3.0489608416090426,
			Memory:    0,
		},
	},
	{
		BeatmapID: 3398802,
		Mods:      osu.HD,
		Skills: Skills{
			Stamina:   227.99391325634804,
			Tenacity:  220.27730795451728,
			Agility:   570.10769571937385,
			Accuracy:  397.24740119334484,
			Precision: 229.3259069817596,
			Reaction:  296.4850383764018,
			Reading:   9.3854347881564912,
			Memory:    0,
		},
	},
	{
		BeatmapID: 3398802,
		Mods:      osu.HR,
		Skills: Skills{
			Stamina:   227.99391325634804,
			Tenacity:  220.27730795451728,
			Agility:   570.10769662603536,
			Accuracy:  668.46235652176449,
			Precision: 387.56078305969339,
			Reaction:  468.82580109131925,
			Reading:   0,
			Memory:    0,
		},
	},
	{
		BeatmapID: 3398802,
		Mods:      osu.FL,
		Skills: Skills{
			Stamina:   227.99391325634804,
			Tenacity:  220.27730795451728,
			Agility:   570.10769571937385,
			Accuracy:  397.24740119334484,
			Precision: 229.3259069817596,
			Reaction:  303.36730582762209,
			Reading:   2.4237296096243619,
			Memory:    603.72827693683132,
		},
	},
	{
		BeatmapID: 1695388,
		Mods:      osu.NM,
		Skills: Skills{
			Stamina:   255.29354781956883,
			Tenacity:  248.14680003266324,
			Agility:   631.04973061269436,
			Accuracy:  464.5690199154655,
			Precision: 265.18326936461318,
			Reaction:  303.62115093805528,
			Reading:   11.123962183139378,
			Memory:    0,
		},
	},
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
