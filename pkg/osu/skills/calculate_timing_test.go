package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/beatmap"
)

// stepTimeout bounds how long a single step is allowed to run for a single
// mod combination before it's reported as hanging. Kept lower than the
// single-mod diagnostic test's 30s since here we're running up to 57
// combinations per file — a lower bound still reliably distinguishes
// "hanging" from "legitimately slow on a marathon map" (see the ms-range
// timings in the previous run) without ballooning total test time.
const stepTimeout = 10 * time.Second

// slowStepThreshold is a logging-only threshold — finished steps taking
// longer than this are called out explicitly, without failing the test.
const slowStepThreshold = 1 * time.Second

// maxHangsPerFile stops scanning further mod combinations for a file once
// this many have hung. If a map hangs on every mod (mod-independent bug,
// e.g. degenerate slider geometry) there's no point burning
// 57 * stepTimeout on a single file once the pattern is already clear;
// this caps the damage while still capturing enough hangs to see whether
// it's mod-specific or universal.
const maxHangsPerFile = 3

var allModCombinations = DefaultModCombinations()

// TestProcessBeatmap_StepTiming_AllMods runs every step of
// ProcessBeatmap for every mod combination against every map in
// test_data, summing each step's duration across mods and reporting which
// specific mod combinations (if any) cause a step to hang — so you can
// tell whether a slow/stuck map is mod-specific (e.g. only DT/HT alter
// timing enough to trigger it) or happens regardless of mods.
func TestProcessBeatmap_StepTiming_AllMods(t *testing.T) {
	files, err := getOsuFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Skip("no .osu files found in ./test_data")
	}

	vars := DefaultVars()

	for _, f := range files {
		f := f
		t.Run(filepath.Base(f), func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			var bm osu.Beatmap
			if err := beatmap.Unmarshal(data, &bm); err != nil {
				t.Fatalf("parse error: %v", err)
			}

			if bm.Mode != osu.ModeOsu {
				t.Skip("skipping non-std map")
			}

			t.Logf("hit_objects=%d mod_combinations=%d", len(bm.HitObjects), len(allModCombinations))

			totals := make(map[string]time.Duration)
			counts := make(map[string]int)     // how many mod combos completed this step, for averaging
			hangs := make(map[string][]string) // step name -> list of mod strings that hung on it

			hangStreak := 0

			for _, mods := range allModCombinations {
				if hangStreak >= maxHangsPerFile {
					t.Logf("skipping remaining mod combinations: %d consecutive hangs already seen for this file", hangStreak)
					break
				}

				modsLabel := modsToString(mods)

				var md *MapData
				stepOK := true

				stepOK = stepOK && runStepAllMods(t, "ApplyMods", modsLabel, totals, counts, hangs, func() {
					modifiedBm := ApplyMods(&bm, mods)
					md = NewMapData(modifiedBm, mods)
				})

				if stepOK {
					stepOK = stepOK && runStepAllMods(t, "PrepareMapData", modsLabel, totals, counts, hangs, func() {
						prepareMapData(md)
					})
				}
				if stepOK {
					stepOK = stepOK && runStepAllMods(t, "CalculateStamina", modsLabel, totals, counts, hangs, func() {
						CalculateStamina(md, vars)
					})
				}
				if stepOK {
					stepOK = stepOK && runStepAllMods(t, "CalculateTenacity", modsLabel, totals, counts, hangs, func() {
						CalculateTenacity(md, vars)
					})
				}
				if stepOK {
					stepOK = stepOK && runStepAllMods(t, "CalculateAimStrains", modsLabel, totals, counts, hangs, func() {
						calculateAimStrains(md, vars)
					})
				}
				if stepOK {
					stepOK = stepOK && runStepAllMods(t, "CalculateAgility", modsLabel, totals, counts, hangs, func() {
						CalculateAgility(md, vars)
					})
				}
				if stepOK {
					stepOK = stepOK && runStepAllMods(t, "CalculatePrecision", modsLabel, totals, counts, hangs, func() {
						CalculatePrecision(md, vars)
					})
				}
				if stepOK {
					stepOK = stepOK && runStepAllMods(t, "CalculateReading", modsLabel, totals, counts, hangs, func() {
						CalculateReading(md, vars, md.HasMod(osu.HD))
					})
				}
				if stepOK {
					stepOK = stepOK && runStepAllMods(t, "CalculateReaction", modsLabel, totals, counts, hangs, func() {
						CalculateReaction(md, vars, md.HasMod(osu.HD))
					})
				}
				if stepOK {
					stepOK = stepOK && runStepAllMods(t, "CalculateMemory", modsLabel, totals, counts, hangs, func() {
						CalculateMemory(md, vars)
					})
				}
				if stepOK {
					stepOK = stepOK && runStepAllMods(t, "CalculateAccuracy", modsLabel, totals, counts, hangs, func() {
						CalculateAccuracy(md, vars)
					})
				}

				if stepOK {
					hangStreak = 0
				} else {
					hangStreak++
				}
			}

			logSummary(t, totals, counts, hangs)
		})
	}
}

// runStepAllMods times fn for a single (step, mod combination) pair. On
// success it accumulates duration into totals/counts. On timeout it
// records the offending mod combination into hangs and returns false
// (without failing the test outright — a hang on one mod combination
// shouldn't stop the scan of the other 56).
func runStepAllMods(
	t *testing.T,
	step, modsLabel string,
	totals map[string]time.Duration,
	counts map[string]int,
	hangs map[string][]string,
	fn func(),
) bool {
	t.Helper()

	start := time.Now()
	done := make(chan struct{})

	go func() {
		defer close(done)
		fn()
	}()

	select {
	case <-done:
		elapsed := time.Since(start)
		totals[step] += elapsed
		counts[step]++
		if elapsed > slowStepThreshold {
			t.Logf("[SLOW] %-22s mods=%-10s took %v", step, modsLabel, elapsed)
		}
		return true
	case <-time.After(stepTimeout):
		hangs[step] = append(hangs[step], modsLabel)
		t.Errorf("[TIMEOUT] %-22s mods=%-10s did not finish within %v", step, modsLabel, stepTimeout)
		return false
	}
}

// logSummary prints, per step, the total and average time across every mod
// combination that completed, sorted slowest-total-first — plus, for any
// step that hung on at least one mod combination, exactly which ones. If a
// step hung on every combination it was attempted on, that's a strong
// signal the issue is in the map itself (e.g. degenerate slider geometry),
// not mod-specific; if it only hung on some, the mods present in that
// subset (and absent from the ones that passed) are the lead to chase.
func logSummary(
	t *testing.T,
	totals map[string]time.Duration,
	counts map[string]int,
	hangs map[string][]string,
) {
	t.Helper()

	type row struct {
		step  string
		total time.Duration
		count int
	}

	rows := make([]row, 0, len(totals))
	for step, total := range totals {
		rows = append(rows, row{step, total, counts[step]})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].total > rows[j].total })

	t.Log("=== summary: total time per step across all completed mod combinations ===")
	for _, r := range rows {
		avg := time.Duration(0)
		if r.count > 0 {
			avg = r.total / time.Duration(r.count)
		}
		t.Logf("  %-22s total=%-12v avg=%-12v completed_mods=%d", r.step, r.total, avg, r.count)
	}

	if len(hangs) == 0 {
		return
	}

	t.Log("=== hangs by step ===")
	for step, modsList := range hangs {
		t.Errorf("  %-22s hung on %d mod combination(s): %s", step, len(modsList), fmt.Sprintf("%v", modsList))
	}
}
