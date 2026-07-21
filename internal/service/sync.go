package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/compico/go-osu/internal/model"
	"github.com/compico/go-osu/internal/repository"
	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/skills"
)

// writeChunkSize is how many rows Syncer hands to a single UpsertBatch call
// at a time. Purely for progress-logging granularity — database.DB still
// further splits a batch internally if a table is too wide for SQLite's
// variable limit; this number can safely stay at 1000 regardless.
const writeChunkSize = 1000

// skillLogEvery throttles per-beatmap skill-calc progress logs so a 16k-map
// sync doesn't emit 16k log lines — only every Nth completed beatmap (plus
// the very first and the very last) gets logged.
const skillLogEvery = 200

// defaultSkillCalcTimeout bounds how long a single beatmap's skill calc
// (across all mod combinations) is allowed to run before it's abandoned.
// A hang here (degenerate slider math, bad timing points, etc.) would
// otherwise permanently occupy a worker slot and eventually stall the
// whole sync once enough workers are stuck on bad maps.
//
// NOTE: this budget is for the *whole* beatmap (all mod combinations
// combined), not per combination. It must comfortably cover 36
// (len(DefaultModCombinations)) sequential skill calculations under
// worst-case CPU contention from sibling workers. If you see timeouts on
// maps that pass in isolation, this is the first thing to check/raise —
// don't "fix" it by growing concurrency instead (see defaultConcurrency).
const defaultSkillCalcTimeout = 30 * time.Second

// defaultConcurrency sets the number of beatmaps processed in parallel.
//
// Skill calculation is CPU-bound (pure math over hit objects/timing
// points, no I/O, no blocking) — every worker actively competes for a
// CPU core the entire time it runs, unlike an I/O-bound worker that
// mostly sits idle waiting on a socket or disk. Oversubscribing a
// CPU-bound pool (running more active workers than cores) doesn't
// increase throughput; it just adds scheduler contention, and — for a
// task with a wall-clock timeout — silently inflates real per-beatmap
// runtime until otherwise-healthy maps start blowing through
// skillCalcTimeout under load, even though they'd finish in a second or
// two in isolation. This was exactly what a previous version of this
// file did with `runtime.NumCPU() * 2`, and it compounded with a
// permanent-elastic-grow semaphore into a feedback loop: more
// contention -> more false-timeout "abandoned" goroutines still burning
// CPU in the background -> even more contention.
//
// GOMAXPROCS (usually == NumCPU) is the right ceiling for a CPU-bound
// pool. We leave one core free for the writer goroutine, watchdog,
// runtime/GC, and the rest of the system, since this typically runs
// for a long time in the background alongside other work.
func defaultConcurrency() int {
	n := runtime.GOMAXPROCS(0) - 1
	if n < 1 {
		n = 1
	}
	return n
}

// watchdogInterval controls how often the in-flight tracker logs beatmaps
// that have been calculating longer than watchdogWarnAfter — lets you spot
// a stuck map live instead of discovering it an hour later.
const watchdogInterval = 2 * time.Second
const watchdogWarnAfter = 5 * time.Second

type Syncer struct {
	osu    *Osu
	repos  *repository.Repos
	logger *slog.Logger

	modCombinations []osu.Mod
	vars            *skills.Vars

	concurrency      int
	skillCalcTimeout time.Duration
}

type SyncerOption func(*Syncer)

func WithConcurrency(n int) SyncerOption {
	return func(s *Syncer) {
		if n > 0 {
			s.concurrency = n
		}
	}
}

func WithSkillCalcTimeout(d time.Duration) SyncerOption {
	return func(s *Syncer) {
		if d > 0 {
			s.skillCalcTimeout = d
		}
	}
}

// NewSyncer requires an explicit logger rather than reaching into Osu's
// internal one — if you don't see sync logs, check what's passed here
// first (nil logger, or one configured below Info level, are the usual
// culprits).
func NewSyncer(osuSvc *Osu, repos *repository.Repos, logger *slog.Logger, opts ...SyncerOption) *Syncer {
	if logger == nil {
		logger = slog.Default()
	}

	s := &Syncer{
		osu:              osuSvc,
		repos:            repos,
		logger:           logger.With("component", "sync"),
		modCombinations:  skills.DefaultModCombinations(),
		vars:             skills.DefaultVars(),
		concurrency:      defaultConcurrency(),
		skillCalcTimeout: defaultSkillCalcTimeout,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Run performs a full sync pass:
//  1. parse osu!.db
//  2. map rows into model.BeatmapSet / model.Beatmap
//  3. diff MD5 hashes against what's already stored to find changed/new maps
//  4. for changed maps only: parse the .osu file and compute Skills for
//     every mod combination via skills.ProcessBeatmap
//  5. persist everything, then drop rows for maps no longer present
func (s *Syncer) Run(ctx context.Context) error {
	runStart := time.Now()
	s.logger.Info("sync started", "concurrency", s.concurrency, "skill_calc_timeout", s.skillCalcTimeout)

	step := time.Now()
	if err := s.osu.ReadOsuDBFile(); err != nil {
		return fmt.Errorf("read osu!.db: %w", err)
	}
	raw := s.osu.GetBeatmaps()
	s.logger.Info("read osu!.db", "beatmaps", len(raw), "took", time.Since(step))

	step = time.Now()
	oldBeatmaps, err := s.repos.Beatmaps.List(ctx)
	if err != nil {
		return fmt.Errorf("load existing beatmaps: %w", err)
	}
	oldByID := make(map[int32]model.Beatmap, len(oldBeatmaps))
	for _, bm := range oldBeatmaps {
		oldByID[bm.BeatmapID] = bm
	}
	s.logger.Info("loaded existing beatmaps from db", "count", len(oldBeatmaps), "took", time.Since(step))

	sets, beatmaps := s.mapRawBeatmaps(raw)

	var changed []osu.DatabaseBeatmap
	for i, bm := range beatmaps {
		old, existed := oldByID[bm.BeatmapID]
		if existed && old.MD5Hash == bm.MD5Hash {
			beatmaps[i].StarsNoMod = old.StarsNoMod
			continue
		}
		changed = append(changed, rawByID(raw, bm.BeatmapID))
	}
	s.logger.Info("diffed beatmaps",
		"total", len(beatmaps),
		"sets", len(sets),
		"changed", len(changed),
		"unchanged", len(beatmaps)-len(changed),
	)

	step = time.Now()
	if err := s.upsertBeatmapSetsBatched(ctx, sets); err != nil {
		return fmt.Errorf("upsert beatmapsets: %w", err)
	}
	s.logger.Info("beatmapsets written", "count", len(sets), "took", time.Since(step))

	if len(changed) > 0 {
		step = time.Now()
		skipped := s.computeChanged(ctx, changed)
		s.logger.Info("skill calc done",
			"beatmaps", len(changed),
			"skipped_timeout", skipped,
			"mod_combinations", len(s.modCombinations),
			"took", time.Since(step),
		)
	} else {
		s.logger.Info("no changed beatmaps, skipping skill calc")
	}

	step = time.Now()
	if err := s.upsertBeatmapsBatched(ctx, beatmaps); err != nil {
		return fmt.Errorf("upsert beatmaps: %w", err)
	}
	s.logger.Info("beatmaps written", "count", len(beatmaps), "took", time.Since(step))

	step = time.Now()
	keepIDs := make([]int32, len(beatmaps))
	for i, bm := range beatmaps {
		keepIDs[i] = bm.BeatmapID
	}
	if err := s.repos.Beatmaps.DeleteMissing(ctx, keepIDs); err != nil {
		return fmt.Errorf("delete missing beatmaps: %w", err)
	}
	s.logger.Info("pruned missing beatmaps", "took", time.Since(step))

	s.logger.Info("sync finished", "took", time.Since(runStart))

	s.logger.Info("goroutine count after sync", "count", runtime.NumGoroutine())

	return nil
}

func (s *Syncer) upsertBeatmapSetsBatched(ctx context.Context, sets []model.BeatmapSet) error {
	total := len(sets)
	for start := 0; start < total; start += writeChunkSize {
		end := start + writeChunkSize
		if end > total {
			end = total
		}
		if err := s.repos.BeatmapSets.UpsertBatch(ctx, sets[start:end]); err != nil {
			return err
		}
		s.logger.Info("beatmapset writing", "progress", fmt.Sprintf("%d/%d", end, total))
	}
	return nil
}

func (s *Syncer) upsertBeatmapsBatched(ctx context.Context, beatmaps []model.Beatmap) error {
	total := len(beatmaps)
	for start := 0; start < total; start += writeChunkSize {
		end := start + writeChunkSize
		if end > total {
			end = total
		}
		if err := s.repos.Beatmaps.UpsertBatch(ctx, beatmaps[start:end]); err != nil {
			return err
		}
		s.logger.Info("beatmap writing", "progress", fmt.Sprintf("%d/%d", end, total))
	}
	return nil
}

func rawByID(raw []osu.DatabaseBeatmap, id int32) osu.DatabaseBeatmap {
	for _, bm := range raw {
		if bm.BeatmapID == id {
			return bm
		}
	}
	return osu.DatabaseBeatmap{}
}

func (s *Syncer) mapRawBeatmaps(raw []osu.DatabaseBeatmap) ([]model.BeatmapSet, []model.Beatmap) {
	setsByID := make(map[int32]model.BeatmapSet)
	beatmaps := make([]model.Beatmap, 0, len(raw))

	for _, bm := range raw {
		if osu.GameMode(bm.Mode) != osu.ModeOsu {
			continue
		}

		if _, ok := setsByID[bm.BeatmapSetID]; !ok {
			setsByID[bm.BeatmapSetID] = model.BeatmapSet{
				BeatmapSetID:  bm.BeatmapSetID,
				SongTitle:     bm.SongTitle,
				SongTitleUni:  bm.SongTitleUni,
				ArtistName:    bm.ArtistName,
				ArtistNameUni: bm.ArtistNameUni,
				CreatorName:   bm.CreatorName,
				SongSource:    bm.SongSource,
				SongTags:      bm.SongTags,
			}
		}

		md5Hash, err := model.ParseMD5Hash(bm.MD5Hash)
		if err != nil {
			s.logger.Warn("failed to parse md5 hash")
			continue
		}

		stars := 0.0
		for _, star := range bm.OsuModeStars {
			if star.Int == 0 {
				stars = float64(star.Float)
			}
		}

		beatmaps = append(beatmaps, model.Beatmap{
			BeatmapID:    bm.BeatmapID,
			BeatmapSetID: bm.BeatmapSetID,

			Difficulty:       bm.Difficulty,
			MD5Hash:          md5Hash,
			FolderName:       bm.FolderName,
			NameOfTheOsuFile: bm.NameOfTheOsuFile,
			AudioFileName:    bm.AudioFileName,
			TitleFont:        bm.TitleFont,

			ApproachRate:      bm.ApproachRate,
			CircleSize:        bm.CircleSize,
			HPDrain:           bm.HPDrain,
			OverallDifficulty: bm.OverallDifficulty,
			SliderVelocity:    bm.SliderVelocity,
			StackLeniency:     bm.StackLeniency,

			BPM:        dominantBPM(bm.TimingPoints, bm.TotalTime),
			StarsNoMod: stars,

			DrainTime:            bm.DrainTime,
			TotalTime:            bm.TotalTime,
			PreviewAudioTime:     bm.PreviewAudioTime,
			ThreadID:             bm.ThreadID,
			LastModificationTime: bm.LastModificationTime,
			LastCheckedOsuRepo:   bm.LastCheckedOsuRepo,
			LastModification:     bm.LastModification,
			LastPlay:             bm.LastPlay,

			NumberOfHitcircles: bm.NumberOfHitcircles,
			NumberOfSliders:    bm.NumberOfSliders,
			NumberOfSpinners:   bm.NumberOfSpinners,
			LocalOffset:        bm.LocalOffset,
			OnlineOffset:       bm.OnlineOffset,

			Mode:               bm.Mode,
			RankedStatus:       bm.RankedStatus,
			GradeAchievedOsu:   bm.GradeAchievedOsu,
			GradeAchievedTaiko: bm.GradeAchievedTaiko,
			GradeAchievedCTB:   bm.GradeAchievedCTB,
			GradeAchievedMania: bm.GradeAchievedMania,
			ManiaScrollSpeed:   bm.ManiaScrollSpeed,

			Unplayed: bm.Unplayed,
		})
	}

	sets := make([]model.BeatmapSet, 0, len(setsByID))
	for _, set := range setsByID {
		sets = append(sets, set)
	}

	return sets, beatmaps
}

func dominantBPM(points []osu.DatabaseTimingPoint, totalTimeMs int32) float64 {
	if len(points) == 0 {
		return 0
	}

	sorted := make([]osu.DatabaseTimingPoint, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].TimeOffset < sorted[j].TimeOffset })

	durationByBPM := make(map[float64]float64)
	currentBPM := 0.0

	for i, p := range sorted {
		if !p.Inherited && p.BeatLength > 0 {
			currentBPM = 60000.0 / p.BeatLength
		}
		if currentBPM <= 0 {
			continue
		}

		end := float64(totalTimeMs)
		if i+1 < len(sorted) {
			end = sorted[i+1].TimeOffset
		}
		if duration := end - p.TimeOffset; duration > 0 {
			durationByBPM[currentBPM] += duration
		}
	}

	var best, bestDuration float64
	for bpm, dur := range durationByBPM {
		if dur > bestDuration {
			bestDuration, best = dur, bpm
		}
	}

	return math.Round(best)
}

// computeChanged runs skill calc for every changed beatmap with bounded,
// static concurrency (s.concurrency workers — sized for CPU-bound work,
// see defaultConcurrency). Each beatmap gets s.skillCalcTimeout to finish
// all mod combinations, enforced via context cancellation that is checked
// *between* mod combinations (computeOneBeatmap), so a slow-but-not-hung
// beatmap actually stops doing work once cancelled instead of merely being
// abandoned by the caller.
//
// The only remaining leak case is a single mod combination that itself
// hangs forever inside skills.ProcessBeatmap — Go has no way to
// force-stop a running goroutine, and we don't control that function's
// internals here. If that happens the inner goroutine keeps running in
// the background for the rest of the process lifetime, same as before.
// With concurrency now matched to actual CPU capacity (instead of 2x
// oversubscribed) this should now be a rare, genuine "this map is
// pathological" case rather than routine false-timeouts caused by
// contention — if you still see repeated timeouts on maps that pass in
// isolation, raise skillCalcTimeout before suspecting this pool again.
//
// A watchdog goroutine logs any beatmap still in flight past
// watchdogWarnAfter every watchdogInterval, so a stuck map shows up in the
// logs live instead of only being discovered once its own timeout fires.
func (s *Syncer) computeChanged(ctx context.Context, changed []osu.DatabaseBeatmap) int64 {
	total := len(changed)

	var done atomic.Int64
	var skipped atomic.Int64

	rows := make(chan model.SkillCache, 4096)
	writerStop := make(chan struct{})
	writerDone := make(chan struct{})
	go s.skillCacheWriter(ctx, rows, writerStop, writerDone)

	inFlight := newInFlightTracker()
	watchdogStop := make(chan struct{})
	go s.watchdog(inFlight, watchdogStop)

	// Static semaphore sized for CPU-bound work — deliberately does NOT
	// grow. Growing the number of *active* CPU-bound workers under load
	// makes contention worse, not better (see defaultConcurrency doc).
	sem := make(chan struct{}, s.concurrency)
	var wg sync.WaitGroup

loop:
	for _, dbBm := range changed {
		select {
		case <-ctx.Done():
			break loop
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(dbBm osu.DatabaseBeatmap) {
			defer wg.Done()
			defer func() { <-sem }()

			inFlight.start(dbBm.BeatmapID, dbBm.FolderName)
			defer inFlight.finish(dbBm.BeatmapID)

			timedOut := s.computeOneBeatmap(ctx, dbBm, rows)
			if timedOut {
				skipped.Add(1)
			}

			s.logSkillProgress(done.Add(1), int64(total))
		}(dbBm)
	}

	wg.Wait()
	close(watchdogStop)

	close(writerStop)
	<-writerDone

	return skipped.Load()
}

// computeOneBeatmap parses the .osu file and computes Skills for every mod
// combination, bailing out after s.skillCalcTimeout. Cancellation is
// checked between mod combinations, so a beatmap that's merely slow (not
// hung) actually stops doing further work once its budget runs out,
// rather than continuing to burn CPU in the background after being
// reported as timed out. Returns true if it timed out.
func (s *Syncer) computeOneBeatmap(parent context.Context, dbBm osu.DatabaseBeatmap, rows chan<- model.SkillCache) bool {
	ctx, cancel := context.WithTimeout(parent, s.skillCalcTimeout)
	defer cancel()

	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)

		full, err := s.osu.ParseBeatmapFile(dbBm)
		if err != nil {
			s.logger.Warn("skipping beatmap, failed to parse .osu file",
				"beatmap_id", dbBm.BeatmapID, "error", err)
			return
		}

		md5Hash, err := model.ParseMD5Hash(dbBm.MD5Hash)
		if err != nil {
			s.logger.Warn("skipping beatmap, invalid md5 hash", "beatmap_id", dbBm.BeatmapID, "error", err)
			return
		}

		for _, mods := range s.modCombinations {
			// Checked between combinations rather than only once up
			// front, so cancellation partway through a beatmap's 36
			// combinations still stops the remaining work instead of
			// running them to completion after the deadline.
			if ctx.Err() != nil {
				return
			}

			result := skills.ProcessBeatmap(full, mods, s.vars)

			select {
			case rows <- model.SkillCache{
				BeatmapID: dbBm.BeatmapID,
				Mods:      int32(mods),
				MD5Hash:   md5Hash,
				Stamina:   result.Skills.Stamina,
				Tenacity:  result.Skills.Tenacity,
				Agility:   result.Skills.Agility,
				Precision: result.Skills.Precision,
				Reading:   result.Skills.Reading,
				Memory:    result.Skills.Memory,
				Accuracy:  result.Skills.Accuracy,
				Reaction:  result.Skills.Reaction,
			}:
			case <-ctx.Done():
				// Caller gave up (or process is shutting down) while we
				// were about to enqueue a row — don't block forever on a
				// full channel, just stop.
				return
			}
		}
	}()

	select {
	case <-doneCh:
		return false
	case <-ctx.Done():
		s.logger.Error("beatmap skill calc timed out, cancelling",
			"beatmap_id", dbBm.BeatmapID,
			"folder", dbBm.FolderName,
			"file", dbBm.NameOfTheOsuFile,
			"timeout", s.skillCalcTimeout,
		)
		return true
	}
}

// skillCacheWriter is the single writer for skill_cache rows, batching
// writes at writeChunkSize. It keeps draining rows after writerStop fires
// until the channel is empty (non-blocking drain), then does a final
// flush.
func (s *Syncer) skillCacheWriter(ctx context.Context, rows <-chan model.SkillCache, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)

	buf := make([]model.SkillCache, 0, writeChunkSize)
	written := 0

	flush := func() {
		if len(buf) == 0 {
			return
		}
		if err := s.repos.Skills.UpsertBatch(ctx, buf); err != nil {
			s.logger.Error("failed to write skill_cache batch", "error", err, "rows", len(buf))
		} else {
			written += len(buf)
			s.logger.Info("skill_cache writing", "written", written)
		}
		buf = buf[:0]
	}

	for {
		select {
		case row := <-rows:
			buf = append(buf, row)
			if len(buf) >= writeChunkSize {
				flush()
			}
		case <-stop:
			for {
				select {
				case row := <-rows:
					buf = append(buf, row)
					if len(buf) >= writeChunkSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}

func (s *Syncer) logSkillProgress(n, total int64) {
	if n == 1 || n == total || n%skillLogEvery == 0 {
		s.logger.Info("skill calc", "progress", fmt.Sprintf("%d/%d", n, total))
	}
}

// inFlightTracker records when each currently-computing beatmap started,
// purely for the watchdog to report on. Safe for concurrent use.
type inFlightTracker struct {
	mu      sync.Mutex
	started map[int32]inFlightEntry
}

type inFlightEntry struct {
	startedAt time.Time
	folder    string
}

func newInFlightTracker() *inFlightTracker {
	return &inFlightTracker{started: make(map[int32]inFlightEntry)}
}

func (t *inFlightTracker) start(beatmapID int32, folder string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.started[beatmapID] = inFlightEntry{startedAt: time.Now(), folder: folder}
}

func (t *inFlightTracker) finish(beatmapID int32) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.started, beatmapID)
}

func (t *inFlightTracker) snapshot() map[int32]inFlightEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make(map[int32]inFlightEntry, len(t.started))
	for k, v := range t.started {
		out[k] = v
	}
	return out
}

// watchdog periodically logs any beatmap that's been calculating longer
// than watchdogWarnAfter, so a stuck map is visible in real time rather
// than only showing up once (or if) its own per-beatmap timeout fires.
func (s *Syncer) watchdog(tracker *inFlightTracker, stop <-chan struct{}) {
	ticker := time.NewTicker(watchdogInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			now := time.Now()
			for beatmapID, entry := range tracker.snapshot() {
				if elapsed := now.Sub(entry.startedAt); elapsed > watchdogWarnAfter {
					s.logger.Warn("beatmap still calculating",
						"beatmap_id", beatmapID,
						"folder", entry.folder,
						"elapsed", elapsed.Round(time.Second),
					)
				}
			}
		}
	}
}
