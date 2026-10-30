package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/beatmap"
	"github.com/compico/go-osu/pkg/osu/database"
)

// TestProcessOsuDatabase_StepTiming is an opt-in diagnostic for a local osu!
// installation. Set OSU_GAME_PATH to the game directory; the test reads
// osu!.db, parses every referenced .osu file, and runs every default mod
// combination. It prints one aggregate timing summary and never prints map
// contents or skill values.
func TestProcessOsuDatabase_StepTiming(t *testing.T) {
	if debug {
		t.Skip("run without the debug build tag to avoid verbose debugf output")
	}

	gamePath := os.Getenv("OSU_GAME_PATH")
	if gamePath == "" {
		t.Skip("set OSU_GAME_PATH to the osu! installation directory")
	}

	totalStart := time.Now()
	databasePath := filepath.Join(gamePath, "osu!.db")
	var db osu.Database
	dbStart := time.Now()
	if err := database.Unmarshal(databasePath, &db); err != nil {
		t.Fatalf("read osu! database %q: %v", databasePath, err)
	}
	timings := newTimingCollector()
	timings.add("read + decode osu!.db", time.Since(dbStart))

	modsList := DefaultModCombinations()
	vars := NewVars()

	workers := runtime.GOMAXPROCS(0) - 1
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan osu.DatabaseBeatmap)
	var workerGroup sync.WaitGroup
	var processed, readErrors, parseErrors, empty atomic.Int64
	var failuresMu sync.Mutex
	var failures []string
	var skippedDetails []string
	workerTimings := make([]*timingCollector, workers)

	for workerIndex := range workers {
		workerTimings[workerIndex] = newTimingCollector()
		workerTiming := workerTimings[workerIndex]
		workerGroup.Add(1)
		go func() {
			defer workerGroup.Done()
			for entry := range jobs {
				path := filepath.Join(gamePath, "Songs", entry.FolderName, entry.NameOfTheOsuFile)
				start := time.Now()
				data, err := os.ReadFile(path)
				workerTiming.add("read .osu file", time.Since(start))
				if err != nil {
					readErrors.Add(1)
					failuresMu.Lock()
					skippedDetails = append(skippedDetails, fmt.Sprintf("read %q: %v", path, err))
					failuresMu.Unlock()
					continue
				}

				var bm osu.Beatmap
				start = time.Now()
				err = beatmap.Unmarshal(data, &bm)
				workerTiming.add("parse .osu file", time.Since(start))
				if err != nil {
					parseErrors.Add(1)
					failuresMu.Lock()
					skippedDetails = append(skippedDetails, fmt.Sprintf("parse %q: %v", path, err))
					failuresMu.Unlock()
					continue
				}

				for _, mods := range modsList {
					if len(bm.HitObjects) == 0 {
						measureStep(workerTiming, "apply mods (empty map)", func() { ApplyMods(&bm, mods) })
						continue
					}
					if failedStep, panicValue := measureSkillSteps(&bm, mods, vars, workerTiming); panicValue != nil {
						failuresMu.Lock()
						failures = append(failures, fmt.Sprintf("beatmap id=%d file=%q mods=%s step=%s: %v",
							entry.BeatmapID, path, mods, failedStep, panicValue))
						failuresMu.Unlock()
						break
					}
				}
				if len(bm.HitObjects) == 0 {
					empty.Add(1)
				} else {
					processed.Add(1)
				}
			}
		}()
	}

	for i := range db.Beatmaps {
		jobs <- db.Beatmaps[i]
	}
	close(jobs)
	workerGroup.Wait()
	for _, workerTiming := range workerTimings {
		for step, aggregate := range workerTiming.byStep {
			current := timings.byStep[step]
			current.total += aggregate.total
			current.count += aggregate.count
			timings.byStep[step] = current
		}
	}

	t.Logf("wall_clock_total=%s osu!.db entries=%d processed=%d read_errors=%d parse_errors=%d empty_maps=%d mod_combinations=%d workers=%d",
		time.Since(totalStart), len(db.Beatmaps), processed.Load(), readErrors.Load(), parseErrors.Load(), empty.Load(), len(modsList), workers)
	if processed.Load() == 0 {
		t.Fatal("no beatmaps were successfully processed")
	}
	logTimingAggregates(t, timings.byStep)
	for _, detail := range skippedDetails {
		t.Logf("skipped map: %s", detail)
	}
	for _, failure := range failures {
		t.Errorf("%s", failure)
	}
}

type timingAggregate struct {
	total time.Duration
	count int
}

type timingCollector struct {
	byStep map[string]timingAggregate
}

func newTimingCollector() *timingCollector {
	return &timingCollector{byStep: make(map[string]timingAggregate)}
}

func (c *timingCollector) add(step string, elapsed time.Duration) {
	current := c.byStep[step]
	current.total += elapsed
	current.count++
	c.byStep[step] = current
}

// measureStep times one synchronous calculation stage. This diagnostic path
// deliberately calls the same underlying methods as ProcessBeatmap without
// adding timers or branches to the library's normal calculation path.
func measureStep(timings *timingCollector, step string, fn func()) {
	start := time.Now()
	fn()
	timings.add(step, time.Since(start))
}

func measureSkillSteps(bm *osu.Beatmap, mods osu.Mod, vars *Vars, timings *timingCollector) (failedStep string, panicValue any) {
	currentStep := ""
	defer func() {
		if recovered := recover(); recovered != nil {
			failedStep = currentStep
			panicValue = recovered
		}
	}()

	runStep := func(step string, fn func()) {
		currentStep = step
		measureStep(timings, step, fn)
	}

	var modified *osu.Beatmap
	runStep("apply mods", func() {
		modified = ApplyMods(bm, mods)
	})
	if len(modified.HitObjects) == 0 {
		// ProcessBeatmap returns a zero result after ApplyMods filters maps
		// containing only spinners/holds, so no later stages run for this mod.
		return "", nil
	}

	var md *MapData
	runStep("create MapData", func() {
		md = NewMapData(modified, mods)
	})

	runStep("prepareTimingPoints", md.prepareTimingPoints)
	runStep("approximateSliderPoints", md.approximateSliderPoints)
	runStep("bakeSliderData", md.bakeSliderData)
	runStep("prepareHitObjectData", md.prepareHitObjectData)
	runStep("gatherTargetPoints", md.gatherTargetPoints)
	runStep("gatherAimAndReadingPoints", md.gatherAimAndReadingPoints)
	runStep("calculateAngles", md.calculateAngles)
	runStep("gatherTapPatterns", md.gatherTapPatterns)
	runStep("calculateAimStrains", func() { md.calculateAimStrains(vars) })
	runStep("calculateTapStrains", func() { md.calculateTapStrains(vars) })
	runStep("calculateReaction", func() { md.calculateReaction(vars, md.HasMod(osu.HD)) })
	runStep("calculateStamina", func() { md.calculateStamina(vars) })
	runStep("calculateTenacity", func() { md.calculateTenacity(vars) })
	runStep("calculateAgility", func() { md.calculateAgility(vars) })
	runStep("calculatePrecision", func() { md.calculatePrecision(vars) })
	runStep("calculateAccuracy", func() { md.calculateAccuracy(vars) })
	if md.HasMod(osu.FL) {
		runStep("calculateMemory", func() { md.calculateMemory(vars) })
	}
	runStep("calculateReading", func() { md.calculateReading(vars, md.HasMod(osu.HD)) })
	return "", nil
}

func logTimingAggregates(t *testing.T, timings map[string]timingAggregate) {
	t.Helper()
	type row struct {
		step  string
		total time.Duration
		count int
	}
	rows := make([]row, 0, len(timings))
	for step, aggregate := range timings {
		rows = append(rows, row{step: step, total: aggregate.total, count: aggregate.count})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].total > rows[j].total })

	t.Log("step timing summary (sum includes all maps and mod combinations):")
	for _, row := range rows {
		average := time.Duration(0)
		if row.count > 0 {
			average = row.total / time.Duration(row.count)
		}
		t.Logf("  %-32s total=%-12s avg=%-12s calls=%d", row.step, row.total, average, row.count)
	}
}
