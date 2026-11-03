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

// A higher approach rate shows the notes later, so there is less time to react.
// It only matters when the notes are further apart than they are visible.
func TestReactionRisesWithApproachRate(t *testing.T) {
	low := reactionOf(alternatingJumps(40, 800, 150, 6))
	high := reactionOf(alternatingJumps(40, 800, 150, 10))
	if !(high > low) {
		t.Errorf("AR10 %v is not above AR6 %v", high, low)
	}
	// Close together, the notes are visible long before they are hit at both.
	a := reactionOf(alternatingJumps(40, 150, 150, 6))
	b := reactionOf(alternatingJumps(40, 150, 150, 10))
	if a != b {
		t.Errorf("fast notes: AR6 %v differs from AR10 %v", a, b)
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
