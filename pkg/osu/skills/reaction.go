package skills

import (
	"math"
	"sort"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

// Reaction models how little of the time an object is visible is left to
// decide where to go.
//
// An object is visible for a while before it is hit (approach rate, Hidden's
// fade): that is the time the player has to see it and decide. Moving to it
// takes part of the gap since the previous object was finished (a slider's
// end, not its start): by Fitts's law the movement time is
// b*log2(1 + distance/diameter), and the distance is weighted by the turn
// angle, a straight continuation is predictable and costs less than a sharp
// change of direction. The larger the share of the gap the movement takes, the
// less of the visible time is left:
//
//	T = visible * exp(-ShareDecay * movement/gap)
//
// and reaction grows as T shrinks:
//
//	value = VerScale * (RefTime / T)^CurveExp
//
// so the approach rate sets the base and the pattern raises it. A fast rhythm
// of short movements costs little, a fast rhythm of long jumps costs a lot.
// An object that comes while the slider before it is still followed needs no
// jump to the slider's end: only the way beyond the follow circle counts, and
// the gap is the time between the two heads.
//
// A slider's end is not where the player leaves it: SliderLeniency ms (map
// time) earlier is enough, and that is added to the gap after a slider.
//
// The share is not capped at 1, so a map that is faster still, DT over NM, is
// harder at every point however impossible it already was. T is never below
// MinTime, a guard only.
// A slider in front of the object keeps the player busy until it ends, so T
// is shortened by SliderPenalty when the object follows a slider and the gap
// is shorter than the visible time.
//
// The first object has no previous one: its T is its visible time. Spinners
// are not targets, but a spinner's end still counts as the end of the previous
// object.

// reactionFollowRadius is the radius of a slider's follow circle in circle
// diameters: 2.4 radii.
const reactionFollowRadius float64 = 1.2

// reactionMaxShare bounds movement/gap for a gap that is 0 or almost.
const reactionMaxShare float64 = 4

// reactionShareKnee is the limit of the saturated share.
const reactionShareKnee float64 = 2

// reactionPoint is the reaction at one hit object.
type reactionPoint struct {
	Time  float64
	Value float64
}

// reactionDecisionTime returns T in milliseconds for an object visible for
// visible ms, hit gap ms after the previous object was finished, travelling
// dist diameters (turn sharpness sharp in [0, 1]).
func reactionDecisionTime(visible, gap, dist, sharp float64, afterSlider bool, vars *Vars) float64 {
	rv := vars.Reaction
	movement := rv.FittsMs * math.Log2(1+dist*(0.5+0.5*sharp))

	// The share of the gap that the movement takes. It can be above 1: a jump
	// that does not fit the gap at all, and a faster map has a larger share.
	share := reactionMaxShare
	if gap > 0 {
		share = math.Min(movement/gap, reactionMaxShare)
	}
	// Beyond a share of 1 a jump is impossible and more of it counts less:
	// the share saturates smoothly towards reactionShareKnee.
	share = reactionShareKnee * share / (reactionShareKnee + share)
	t := visible * math.Exp(-rv.ShareDecay*share)
	if afterSlider && gap < visible {
		t *= 1 - rv.SliderPenalty
	}
	return math.Max(t, rv.MinTime)
}

// reactionAt turns a decision time into a skill value.
func reactionAt(decision float64, vars *Vars) float64 {
	return vars.Reaction.VerScale * math.Pow(vars.Reaction.RefTime/decision, vars.Reaction.CurveExp)
}

// reactionPoints returns the reaction at every hit object that is a target.
func (md *MapData) reactionPoints(vars *Vars, hidden bool) []reactionPoint {
	hos := md.Map.HitObjects
	ar := md.Map.ApproachRate
	diameter := 2 * float64(cs2px(md.Map.CircleSize))
	fade := vars.Reaction.FadeinPercent

	points := make([]reactionPoint, 0, len(hos))
	prevTime := math.MinInt32
	for i := range hos {
		ho := &hos[i]
		// Objects a few milliseconds after another cannot be hit separately.
		if absInt(ho.Time-prevTime) < 5 {
			continue
		}
		prevTime = ho.Time
		if ho.Type.IsHitSpinner() {
			continue
		}

		visible := float64(ho.Time - getVisibilityTimes(*ho, ar, hidden, fade, 1.0).first)
		if visible <= 0 {
			continue
		}

		decision := visible
		if i > 0 {
			prev := &hos[i-1]
			gap := float64(ho.Time - endTimeOf(prev))
			from := endPointOf(prev)
			dist := from.DistanceFrom(ho.Pos) / diameter
			if gap < 0 && prev.Type.IsHitSlider() {
				// The object comes while the slider is still being followed: the
				// cursor stays inside the follow circle, so only the way beyond it
				// is a movement, and the time between the two heads is what there is.
				gap = float64(ho.Time - prev.Time)
				dist = math.Max(0, dist-reactionFollowRadius)
			} else if endTimeOf(prev) > prev.Time {
				// A slider can be let go a little before its end; the leniency is
				// in map time, so the time of the mod's speed is shorter.
				gap += vars.Reaction.SliderLeniency / modsSpeed(md.Mods)
			}

			sharp := 0.5 // no turn to judge: assume a middling one
			if i > 1 {
				sharp = (math.Pi - getAngle(endPointOf(&hos[i-2]), endPointOf(prev), ho.Pos)) / math.Pi
				if math.IsNaN(sharp) {
					sharp = 0.5
				}
			}
			afterSlider := endTimeOf(prev) > prev.Time
			decision = reactionDecisionTime(visible, gap, dist, sharp, afterSlider, vars)
		}

		points = append(points, reactionPoint{Time: float64(ho.Time), Value: reactionAt(decision, vars)})
	}
	return points
}

// CalculateReaction fills md.Skills.Reaction.
func CalculateReaction(md *MapData, vars *Vars, hidden bool) {
	md.calculateReaction(vars, hidden)
}

func (md *MapData) calculateReaction(vars *Vars, hidden bool) {
	md.Skills.Reaction = reactionSkill(md.reactionValues(vars, hidden), vars)
}

// reactionValues returns the reaction skill at every target.
func (md *MapData) reactionValues(vars *Vars, hidden bool) []float64 {
	points := md.reactionPoints(vars, hidden)
	values := make([]float64, len(points))
	for i, p := range points {
		values[i] = p.Value
	}
	return values
}

// reactionSkill is the mean of the strongest values: the top TopShare of all
// values, at least reactionMinTop of them. One note cannot make a map hard to
// react to, a passage can; the mean of the top few is that passage.
func reactionSkill(values []float64, vars *Vars) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Sort(sort.Reverse(sort.Float64Slice(sorted)))

	n := int(math.Ceil(float64(len(sorted)) * vars.Reaction.TopShare))
	n = min(max(n, reactionMinTop), len(sorted))

	sum := 0.0
	for _, v := range sorted[:n] {
		sum += v
	}
	return sum / float64(n)
}

// reactionMinTop is the fewest values reactionSkill averages.
const reactionMinTop = 5

// endTimeOf is when the object is finished: a slider's end, otherwise its time.
func endTimeOf(ho *osu.HitObject) int {
	return max(ho.EndTime, ho.Time)
}

// endPointOf is where the cursor is after the object: a slider's end,
// otherwise its position.
func endPointOf(ho *osu.HitObject) vector2d.Vector2dd {
	if ho.Type.IsHitSlider() {
		return ho.EndPoint
	}
	return ho.Pos
}
