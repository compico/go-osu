package skills

import (
	"math"
	"testing"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

// alternatingJumps builds n circles that alternate between two places dist px
// apart, interval ms apart.
func alternatingJumps(n int, interval int, dist float64, ar float64) *osu.Beatmap {
	bm := circlesAt()
	bm.ApproachRate = ar
	for i := 0; i < n; i++ {
		x := 100.0
		if i%2 == 1 {
			x += dist
		}
		bm.HitObjects = append(bm.HitObjects, osu.HitObject{
			Pos:  vector2d.Vector2dd{X: x, Y: 200},
			Time: 1000 + i*interval,
			Type: osu.HitNormal,
		})
	}
	return bm
}

func reactionOf(bm *osu.Beatmap) float64 {
	return ProcessBeatmap(bm, osu.NM, NewVars()).Skills.Reaction
}

func TestReactionDecisionTimeShrinks(t *testing.T) {
	vars := NewVars()
	base := reactionDecisionTime(500, 300, 2, 0.5, false, vars)

	cases := []struct {
		name string
		got  float64
	}{
		{"shorter gap", reactionDecisionTime(500, 150, 2, 0.5, false, vars)},
		{"longer jump", reactionDecisionTime(500, 300, 4, 0.5, false, vars)},
		{"sharper turn", reactionDecisionTime(500, 300, 2, 1, false, vars)},
		{"shorter visibility", reactionDecisionTime(200, 300, 2, 0.5, false, vars)},
		{"after a slider", reactionDecisionTime(500, 300, 2, 0.5, true, vars)},
	}
	for _, c := range cases {
		if !(c.got < base) {
			t.Errorf("%s: decision time %v is not below %v", c.name, c.got, base)
		}
	}
}

func TestReactionDecisionTimeIsBounded(t *testing.T) {
	vars := NewVars()
	if got := reactionDecisionTime(500, 1, 8, 1, true, vars); got != vars.Reaction.MinTime {
		t.Errorf("impossible note: %v, want MinTime %v", got, vars.Reaction.MinTime)
	}
	// A slider only costs time when the gap is shorter than the visibility.
	if a, b := reactionDecisionTime(300, 600, 1, 0.5, true, vars), reactionDecisionTime(300, 600, 1, 0.5, false, vars); a != b {
		t.Errorf("slider before a slow note: %v vs %v", a, b)
	}
}

func TestReactionAtIsTheReferenceAtRefTime(t *testing.T) {
	vars := NewVars()
	if got := reactionAt(vars.Reaction.RefTime, vars); math.Abs(got-vars.Reaction.VerScale) > 1e-9 {
		t.Errorf("got %v, want %v", got, vars.Reaction.VerScale)
	}
	if !(reactionAt(100, vars) > reactionAt(200, vars)) {
		t.Error("less time must be more reaction")
	}
}

// Fast alternating jumps are harder to react to than slow ones, at any AR.
func TestReactionPrefersFastJumps(t *testing.T) {
	for _, ar := range []float64{7, 9, 10} {
		fast := reactionOf(alternatingJumps(40, 110, 110, ar))
		slow := reactionOf(alternatingJumps(40, 400, 110, ar))
		if !(fast > slow) {
			t.Errorf("AR%v: fast %v is not above slow %v", ar, fast, slow)
		}
	}
}

// At the same speed, longer jumps need more reaction than short ones.
func TestReactionPrefersLongJumps(t *testing.T) {
	long := reactionOf(alternatingJumps(40, 150, 250, 9))
	short := reactionOf(alternatingJumps(40, 150, 40, 9))
	if !(long > short) {
		t.Errorf("long %v is not above short %v", long, short)
	}
}

// A higher approach rate shows the notes later, so there is less time to react,
// at any speed.
func TestReactionRisesWithApproachRate(t *testing.T) {
	for _, interval := range []int{150, 300, 800} {
		low := reactionOf(alternatingJumps(40, interval, 150, 6))
		high := reactionOf(alternatingJumps(40, interval, 150, 10))
		if !(high > low) {
			t.Errorf("interval %d: AR10 %v is not above AR6 %v", interval, high, low)
		}
	}
}

// The share of the gap that the movement takes is what raises the reaction:
// the same fast rhythm is harder with long jumps than with short ones, and an
// easy rhythm stays near the approach rate's base.
func TestReactionDecisionTimeIsTheVisibleTimeWhenThereIsNoMovement(t *testing.T) {
	vars := NewVars()
	if got := reactionDecisionTime(500, 400, 0, 0, false, vars); got != 500 {
		t.Errorf("no movement: %v, want the visible time 500", got)
	}
	// Half of the gap spent on the movement: share 0.5, saturated by the knee.
	gap := 2 * vars.Reaction.FittsMs // dist 1, sharpness 1: one bit
	share := reactionShareKnee * 0.5 / (reactionShareKnee + 0.5)
	want := 500 * math.Exp(-vars.Reaction.ShareDecay*share)
	if got := reactionDecisionTime(500, gap, 1, 1, false, vars); math.Abs(got-want) > 1e-9 {
		t.Errorf("half the gap: %v, want %v", got, want)
	}
}

// A mod that makes the map faster makes it harder at every point, however
// impossible the point already was.
func TestReactionRisesWithSpeedEvenWhenImpossible(t *testing.T) {
	vars := NewVars()
	// 5 diameters in the gap that one needs about 100 ms for: share 1.5 and 3.
	slow := reactionDecisionTime(500, 70, 5, 1, false, vars)
	fast := reactionDecisionTime(500, 35, 5, 1, false, vars)
	if !(fast < slow) {
		t.Errorf("a shorter gap gives %v, not below %v", fast, slow)
	}
}

// A slider can be let go a little before its end, so the gap after it is
// longer by SliderLeniency in map time: less of it in real time under DT.
func TestReactionAfterSliderUsesTheLeniency(t *testing.T) {
	bm := circlesAt()
	bm.ApproachRate = 9
	at := func(x float64, tm int, typ osu.HitObjectType, end int) osu.HitObject {
		p := vector2d.Vector2dd{X: x, Y: 200}
		return osu.HitObject{Pos: p, EndPoint: p, Time: tm, EndTime: end, Type: typ}
	}
	bm.HitObjects = []osu.HitObject{
		at(100, 1000, osu.HitNormal, 1000),
		at(100, 1400, osu.HitSlider, 1500),
		at(300, 1600, osu.HitNormal, 1600),
	}

	last := func(mods osu.Mod, leniency float64) float64 {
		vars := NewVars()
		vars.Reaction.SliderLeniency = leniency
		// reactionPoints needs only the hit objects, not the prepared slider data.
		md := NewMapData(ApplyMods(bm, mods), mods)
		p := md.reactionPoints(vars, false)
		return p[len(p)-1].Value
	}

	if none, some := last(osu.NM, 0), last(osu.NM, 36); !(some < none) {
		t.Errorf("leniency: %v is not below %v", some, none)
	}
	// The same leniency is worth less real time under DT.
	nm := last(osu.NM, 0) - last(osu.NM, 36)
	dt := last(osu.DT, 0) - last(osu.DT, 36)
	if nm <= 0 || dt <= 0 {
		t.Errorf("leniency does not lower the reaction: NM %v, DT %v", nm, dt)
	}
}

// reactionSkill is the mean of the strongest values.
func TestReactionSkillOfValues(t *testing.T) {
	vars := NewVars()
	if got := reactionSkill(nil, vars); got != 0 {
		t.Errorf("no values: %v, want 0", got)
	}
	if got := reactionSkill([]float64{3, 9}, vars); got != 6 {
		t.Errorf("fewer values than the minimum: %v, want their mean 6", got)
	}
	// 100 values: the top 5% is 5 values, the 5 largest.
	values := make([]float64, 100)
	for i := range values {
		values[i] = float64(i)
	}
	if got, want := reactionSkill(values, vars), (99.0+98+97+96+95)/5; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	// 200 values: the top 5% is 10 values.
	values = make([]float64, 200)
	for i := range values {
		values[i] = float64(i)
	}
	if got, want := reactionSkill(values, vars), (199.0+190)/2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// One note cannot move the skill by more than its share.
func TestReactionSkillIgnoresOneOutlier(t *testing.T) {
	vars := NewVars()
	values := make([]float64, 100)
	for i := range values {
		values[i] = 100
	}
	base := reactionSkill(values, vars)
	values[50] = 10000
	if got := reactionSkill(values, vars); got > base*30 || got < base {
		t.Errorf("one outlier: %v from %v", got, base)
	}
	if got := reactionSkill(values, vars); got != (10000.0+400)/5 {
		t.Errorf("got %v", got)
	}
}

func TestReactionSkipsSpinners(t *testing.T) {
	bm := circlesAt(1000, 1400, 2000)
	bm.HitObjects[1].Type = osu.HitSpinner
	bm.HitObjects[1].EndTime = 1800

	md := NewMapData(bm, osu.NM)
	md.prepare()
	points := md.reactionPoints(NewVars(), false)
	if len(points) != 2 || points[0].Time != 1000 || points[1].Time != 2000 {
		t.Errorf("points = %+v, want the two circles", points)
	}
}

// An object that comes while the slider before it is still followed needs no
// jump: it is inside the follow circle. It must not be read as a jump over the
// few milliseconds between the slider's end and the object.
func TestReactionInsideTheFollowCircleIsNoJump(t *testing.T) {
	at := func(x float64, tm int, typ osu.HitObjectType, end int) osu.HitObject {
		p := vector2d.Vector2dd{X: x, Y: 200}
		return osu.HitObject{Pos: p, EndPoint: p, Time: tm, EndTime: end, Type: typ}
	}
	last := func(x float64, tm int) float64 {
		bm := circlesAt()
		bm.ApproachRate = 9
		bm.HitObjects = []osu.HitObject{
			at(100, 1000, osu.HitNormal, 1000),
			at(100, 1400, osu.HitSlider, 1500),
			at(x, tm, osu.HitNormal, tm),
		}
		md := NewMapData(bm, osu.NM)
		p := md.reactionPoints(NewVars(), false)
		return p[len(p)-1].Value
	}

	// 60 px is within the follow circle (CS4: a diameter is about 73 px), 10 ms
	// before the slider's end; the same place 10 ms after the end is a jump.
	inside, after := last(160, 1490), last(160, 1510)
	if !(inside < after) {
		t.Errorf("inside the follow circle %v is not below the jump %v", inside, after)
	}
	// With no way beyond the follow circle the base is the visible time, with
	// the slider's penalty only.
	if inside >= reactionAt(250, NewVars()) {
		t.Errorf("inside the follow circle: %v, want a low value", inside)
	}
}
