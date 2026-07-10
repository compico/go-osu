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
	"github.com/compico/go-osu/pkg/dynamic_semaphore"
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
const defaultSkillCalcTimeout = 10 * time.Second

const defaultElasticGrowAfter = 10 * time.Second

// defaultConcurrency intentionally undercuts runtime.NumCPU() — running
// fewer beatmaps in parallel means a stuck one blocks fewer workers at
// once and leaves headroom for the rest of the system while a multi-hour
// sync is running.
func defaultConcurrency() int {
	n := runtime.NumCPU() * 2
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
	elasticGrowAfter time.Duration
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

func WithElasticGrowAfter(d time.Duration) SyncerOption {
	return func(s *Syncer) {
		if d > 0 {
			s.elasticGrowAfter = d
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
		elasticGrowAfter: defaultElasticGrowAfter,
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
//     every mod combination via skills.CalculateAllSkillsForMods
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
		starRatings, skipped := s.computeChanged(ctx, changed)
		s.logger.Info("skill calc done",
			"beatmaps", len(changed),
			"skipped_timeout", skipped,
			"mod_combinations", len(s.modCombinations),
			"took", time.Since(step),
		)

		for i, bm := range beatmaps {
			if sr, ok := starRatings[bm.BeatmapID]; ok {
				beatmaps[i].StarsNoMod = sr
			}
		}
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

		beatmaps = append(beatmaps, model.Beatmap{
			BeatmapID:    bm.BeatmapID,
			BeatmapSetID: bm.BeatmapSetID,

			Difficulty:       bm.Difficulty,
			MD5Hash:          md5Hash,
			FolderName:       bm.FolderName,
			NameOfTheOsuFile: bm.NameOfTheOsuFile,
			AudioFileName:    bm.AudioFileName,
			TitleFont:        bm.TitleFont,

			ApproachRate:      float64(bm.ApproachRate),
			CircleSize:        float64(bm.CircleSize),
			HPDrain:           float64(bm.HPDrain),
			OverallDifficulty: float64(bm.OverallDifficulty),
			SliderVelocity:    bm.SliderVelocity,
			StackLeniency:     float64(bm.StackLeniency),

			BPM:        dominantBPM(bm.TimingPoints, bm.TotalTime),
			StarsNoMod: 0,

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

// computeChanged runs skill calc for every changed beatmap with bounded
// concurrency (s.concurrency workers). Each beatmap gets s.skillCalcTimeout
// to finish all mod combinations; if it doesn't, the worker slot is freed
// and the sync moves on — the underlying goroutine is abandoned (leaked)
// rather than killed, since Go has no way to force-stop a running
// goroutine. This is safe: it never writes to a closed channel (rows is
// never closed) and never panics, it just wastes CPU in the background
// for the remainder of the process lifetime.
//
// A watchdog goroutine logs any beatmap still in flight past
// watchdogWarnAfter every watchdogInterval, so a stuck map shows up in the
// logs live instead of only being discovered once its own timeout fires.
func (s *Syncer) computeChanged(ctx context.Context, changed []osu.DatabaseBeatmap) (map[int32]float64, int64) {
	total := len(changed)

	starRatings := make(map[int32]float64, total)
	var starMu sync.Mutex

	var done atomic.Int64
	var skipped atomic.Int64
	var grown atomic.Int64

	rows := make(chan model.SkillCache, 4096)
	writerStop := make(chan struct{})
	writerDone := make(chan struct{})
	go s.skillCacheWriter(ctx, rows, writerStop, writerDone)

	inFlight := newInFlightTracker()
	watchdogStop := make(chan struct{})
	go s.watchdog(inFlight, watchdogStop)

	sem := dynamic_semaphore.NewDynamicSemaphore(s.concurrency)
	var wg sync.WaitGroup

	for _, dbBm := range changed {
		select {
		case <-ctx.Done():
			goto drain
		default:
		}

		sem.Acquire()
		wg.Add(1)
		go func(dbBm osu.DatabaseBeatmap) {
			defer wg.Done()

			inFlight.start(dbBm.BeatmapID, dbBm.FolderName)
			defer inFlight.finish(dbBm.BeatmapID)

			// elasticGrowTimer: if this beatmap is still running past
			// elasticGrowAfter, permanently open up one extra slot so a
			// short map queued behind it isn't stuck waiting on it —
			// this is exactly the marathon-blocks-the-queue case.
			growTimer := time.AfterFunc(s.elasticGrowAfter, func() {
				sem.Grow(1)
				grown.Add(1)
				s.logger.Warn("beatmap running long, growing concurrency by 1",
					"beatmap_id", dbBm.BeatmapID,
					"after", s.elasticGrowAfter,
				)
			})

			timedOut := s.computeOneBeatmap(dbBm, rows, &starMu, starRatings)

			growTimer.Stop() // no-op if it already fired — grow is permanent either way
			sem.Release()

			if timedOut {
				skipped.Add(1)
			}

			s.logSkillProgress(done.Add(1), int64(total))
		}(dbBm)
	}

drain:
	wg.Wait()
	close(watchdogStop)

	close(writerStop)
	<-writerDone

	s.logger.Info("elastic concurrency growth", "times_grown", grown.Load())

	return starRatings, skipped.Load()
}

// computeOneBeatmap parses the .osu file and computes Skills for every mod
// combination, bailing out after s.skillCalcTimeout. Returns true if it
// timed out (the inner goroutine is left running in the background).
func (s *Syncer) computeOneBeatmap(dbBm osu.DatabaseBeatmap, rows chan<- model.SkillCache, starMu *sync.Mutex, starRatings map[int32]float64) bool {
	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)

		full, err := s.osu.ParseBeatmapFile(dbBm)
		if err != nil {
			s.logger.Warn("skipping beatmap, failed to parse .osu file",
				"beatmap_id", dbBm.BeatmapID, "error", err)
			return
		}

		for _, mods := range s.modCombinations {
			result, err := skills.CalculateAllSkillsForMods(full, mods, s.vars)
			if err != nil {
				s.logger.Warn("skill calc failed for beatmap",
					"beatmap_id", dbBm.BeatmapID, "mods", mods, "error", err)
				continue
			}

			if mods == 0 {
				starMu.Lock()
				starRatings[dbBm.BeatmapID] = aggregateStarRating(result.Skills)
				starMu.Unlock()
			}

			md5Hash, err := model.ParseMD5Hash(dbBm.MD5Hash)
			if err != nil {
				s.logger.Warn("skipping beatmap, invalid md5 hash", "beatmap_id", dbBm.BeatmapID, "error", err)
				continue
			}

			// rows is never closed, so this is safe to block on even if
			// the outer call already gave up and moved on.
			rows <- model.SkillCache{
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
			}
		}
	}()

	select {
	case <-doneCh:
		return false
	case <-time.After(s.skillCalcTimeout):
		s.logger.Error("beatmap skill calc timed out, abandoning and moving on",
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
// flush — this lets any already-queued rows from normal (non-timed-out)
// beatmaps land, while abandoned/leaked goroutines that keep trying to
// send afterward simply block forever on their own, harmlessly.
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

func aggregateStarRating(sk skills.Skills) float64 {
	avg := (sk.Stamina + sk.Tenacity + sk.Agility + sk.Precision +
		sk.Reading + sk.Memory + sk.Accuracy + sk.Reaction) / 8

	k := 0.1

	return 20 * (1 - math.Exp(-k*avg))
}
