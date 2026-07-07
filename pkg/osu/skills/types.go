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

	Mods       osu.Mod
	ModsString string

	// TimeMapper is a helper for using a hit object's time as an index
	// into Map.HitObjects (ported from Beatmap.timeMapper).
	TimeMapper map[int]int

	Spinners int

	// Aim
	Velocities       Velocities
	Distances        []float64
	AimStrains       []float64
	AngleStrains     []float64
	PrecisionStrains []float64
	AimPoints        []AimPoint
	AccuracyStrains  []float64

	// Reading
	Angles          []float64
	AngleBonuses    []float64
	ReactionTimes   []int
	ReactionStrains []float64
	ReadingStrains  []float64
	ReadingPoints   []ReadingPoint
	MemoryStrains   []float64

	// Tapping
	PressIntervals []int
	TapStrains     []float64

	// Tenacity / Stamina
	Streams         map[int][]Stream
	Bursts          map[int][]Burst
	TenacityStrains []float64

	Patterns Patterns
	Skills   Skills
}

// NewMapData creates an independent calculation context for a beatmap.
// bm should be treated as read-only from this point on if it (or copies of
// this MapData) will be used from more than one goroutine.
func NewMapData(bm *osu.Beatmap, mods osu.Mod) *MapData {
	return &MapData{
		Map:        bm,
		Mods:       mods,
		TimeMapper: make(map[int]int, len(bm.HitObjects)),
		Streams:    make(map[int][]Stream),
		Bursts:     make(map[int][]Burst),
	}
}

// HasMod ports the C++ HasMod(beatmap, mod) helper as a method.
func (md *MapData) HasMod(m osu.Mod) bool {
	return md.Mods&m != 0
}
