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

	// TimeMapper is a helper for using a hit object's time as an index
	// into Map.HitObjects (ported from Beatmap.timeMapper).
	TimeMapper map[int]int

	// Aim
	Velocities Velocities
	Distances  []float64
	AimStrains []float64
	AimPoints  []AimPoint

	// Reading
	Angles         []float64
	AngleBonuses   []float64
	ReadingStrains []float64
	ReadingPoints  []ReadingPoint

	// Tapping
	PressIntervals []int
	TapStrains     []float64

	// Tenacity / Stamina
	Streams map[int][][]int
	Bursts  map[int][][]int

	TargetPoints []TargetPoint

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
		TimeMapper: make(map[int]int, len(bm.HitObjects)),
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
