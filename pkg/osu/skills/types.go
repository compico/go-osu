package skills

import (
	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

// AimPointType mirrors globals.h's AIM_POINT_TYPES enum.
type AimPointType int

const (
	AimPointNone AimPointType = iota
	AimPointCircle
	AimPointSlider
	AimPointSliderReverse
	AimPointSliderEnd
)

// AimPoint mirrors globals.h's AimPoint struct.
type AimPoint struct {
	Time int
	Pos  vector2d.Vector2dd
	Type AimPointType
}

type TimingPoint struct {
	Offset       int
	BeatInterval float64
	Meter        int
	Inherited    bool
	Sm           float64
	Bpm          float64
}

// TargetPoint mirrors reaction.cpp/generic.cpp's TIMING struct — a point in
// time the player must react to and hit (either a circle's start, or a
// slider tick).
type TargetPoint struct {
	Time  float64
	Pos   vector2d.Vector2dd
	Key   int  // index used to look back into Map.HitObjects
	Press bool // true for slider ticks, false for circle starts
}

// ReadingPoint mirrors globals.h's ReadingPoint struct.
type ReadingPoint struct {
	Index   int
	Time    int
	Preempt int
	FadeIn  int
	FadeOut int
	Angle   float64
	Pos     vector2d.Vector2dd
	Dist    float64
}

// Burst mirrors globals.h's Burst struct (used by Stamina).
type Burst struct {
	Interval int
	Strain   float64
}

// Stream mirrors globals.h's Stream struct (used by Tenacity).
type Stream struct {
	Interval int
	Length   int
}

// Velocities mirrors the anonymous "velocities" struct nested in the C++
// Beatmap.
type Velocities struct {
	X, Y, Xchange, Ychange []float64
}

// MovementData contains legacy movement metrics.
//
// Deprecated: the skill pipeline no longer calculates these unused metrics.
type MovementData struct {
	Distances  []float64
	Velocities Velocities
}

// Patterns mirrors the anonymous "patterns" struct nested in the C++
// Beatmap.
type Patterns struct {
	CompressedStream []int
	Stream           []int
	Stack            []int
}

// Skills mirrors globals.h's Skills struct: the final calculated skill
// point values.
type Skills struct {
	Stamina   float64
	Tenacity  float64
	Agility   float64
	Precision float64
	Reading   float64
	Memory    float64
	Accuracy  float64
	Reaction  float64
}

// MapData holds every piece of state that the original C++ Beatmap struct
// carried purely for skill-calculation purposes (as opposed to state that
// describes the beatmap file itself, which lives in *osu.Beatmap).
//
// Thread-safety: MapData deliberately does NOT duplicate osu.Beatmap's
// fields. Instead it holds a pointer to one. The intended usage pattern is:
//
//   - Parse a *osu.Beatmap once, then treat it as read-only.
//   - For every calculation you want to run concurrently (different mod
//     combinations, different maps, etc.), create your own *MapData via
//     NewMapData.
//   - Never share a single *MapData across goroutines that might write to
//     it at the same time; a single *MapData is only safe for sequential
//     use (or concurrent *read-only* use once calculation has finished).
//
// Under this pattern, no mutex is needed anywhere in the skills package:
// each goroutine's *MapData is its own private, unshared piece of state,
// and the *osu.Beatmap it points to is never mutated after parsing.
type MapData struct {
	Map *osu.Beatmap

	Mods osu.Mod

	TimingPoints []TimingPoint

	// Aim
	// Deprecated: ProcessBeatmap no longer calculates these unused metrics.
	Velocities Velocities
	// Deprecated: ProcessBeatmap no longer calculates these unused metrics.
	Distances  []float64
	AimStrains []float64
	// AimStrainTimes[i] is the time of the aim point AimStrains[i] belongs to.
	// calculateAimStrains skips points that share a time with the previous
	// one, so AimStrains can be shorter than AimPoints and its indexes do not
	// line up with it.
	AimStrainTimes []int
	AimPoints      []AimPoint

	// Reading
	AngleBonuses   []float64
	ReadingStrains []float64
	ReadingPoints  []ReadingPoint

	// Tapping
	PressIntervals []int
	// PressTimes[i] is the time of the press that ends PressIntervals[i].
	PressTimes []int
	TapStrains []float64
	// TapStrainTimes[i] is the time of the press TapStrains[i] belongs to.
	// TapStrains skips very short intervals, so it can be shorter than
	// PressIntervals and its indexes do not line up with it.
	TapStrainTimes []int

	// Tenacity / Stamina
	Streams map[int][][]int
	Bursts  map[int][][]int

	TargetPoints []TargetPoint

	// Memory: MemoryTotals[i] is the total of memory points up to and
	// including the hit object at MemoryTimes[i]. Spinners and the first hit
	// object have no entry. Only filled by calculateMemory.
	MemoryTotals []float64
	MemoryTimes  []int

	Skills Skills

	BpmMin float64
	BpmMax float64
}

// NewMapData creates an independent calculation context for a beatmap.
// bm should be treated as read-only from this point on if it (or copies of
// this MapData) will be used from more than one goroutine.
func NewMapData(bm *osu.Beatmap, mods osu.Mod) *MapData {
	return &MapData{
		Map:        bm,
		Mods:       mods,
		Streams:    make(map[int][][]int),
		Bursts:     make(map[int][][]int),
		TapStrains: make([]float64, 0),
	}
}

// HasMod ports the C++ HasMod(beatmap, mod) helper as a method.
func (md *MapData) HasMod(m osu.Mod) bool {
	return md.Mods&m != 0
}

func (md *MapData) HasAnyMods() bool {
	return md.Mods != 0
}
