package skills

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/beatmap"
	"github.com/compico/go-osu/pkg/vector2d"
)

// curvesTestBeatmap parses a generated .osu file that exercises every branch of
// the tap strain calculation: a stream, notes only 10 ms apart (skipped by the
// strain), sliders of every curve type with repeats, a pause longer than
// Stamina.LargestInterval and jumps.
func curvesTestBeatmap(t testing.TB) *osu.Beatmap {
	t.Helper()

	var b strings.Builder
	b.WriteString(`osu file format v14

[General]
Mode: 0

[Difficulty]
HPDrainRate:5
CircleSize:4
OverallDifficulty:8
ApproachRate:9
SliderMultiplier:1.4
SliderTickRate:1

[TimingPoints]
0,500,4,2,1,60,1,0
6000,-50,4,2,1,60,0,0

[HitObjects]
`)

	n, now := 0, 1000
	pos := func() (int, int) {
		n++
		return 40 + (n*97)%560, 40 + (n*53)%400
	}
	circle := func() {
		x, y := pos()
		fmt.Fprintf(&b, "%d,%d,%d,1,0,0:0:0:0:\n", x, y, now)
	}

	for i := 0; i < 40; i++ { // stream
		circle()
		now += 75
	}
	for i := 0; i < 4; i++ { // pairs of notes 10 ms apart
		now += 400
		circle()
		now += 10
		circle()
	}
	now += 600
	for i := 0; i < 12; i++ { // sliders
		x, y := pos()
		x2, y2 := pos()
		x3, y3 := pos()
		slides, length := 1+i%3, 90+i*20
		switch i % 3 {
		case 0:
			fmt.Fprintf(&b, "%d,%d,%d,2,0,L|%d:%d,%d,%d\n", x, y, now, x2, y2, slides, length)
		case 1:
			fmt.Fprintf(&b, "%d,%d,%d,2,0,B|%d:%d|%d:%d,%d,%d\n", x, y, now, x2, y2, x3, y3, slides, length)
		default:
			fmt.Fprintf(&b, "%d,%d,%d,2,0,P|%d:%d|%d:%d,%d,%d\n", x, y, now, x2, y2, x3, y3, slides, length)
		}
		now += 1500
	}
	now += 51000 // longer than Stamina.LargestInterval
	for i := 0; i < 10; i++ {
		circle()
		now += 150
	}
	for i := 0; i < 30; i++ { // jumps
		circle()
		now += 300
	}

	bm := &osu.Beatmap{}
	if err := beatmap.Unmarshal([]byte(b.String()), bm); err != nil {
		t.Fatalf("parse generated beatmap: %v", err)
	}
	if len(bm.HitObjects) == 0 {
		t.Fatal("generated beatmap has no hit objects")
	}

	return bm
}

// circlesAt builds a beatmap of circles at the given times.
func circlesAt(times ...int) *osu.Beatmap {
	bm := &osu.Beatmap{
		ApproachRate:      8,
		OverallDifficulty: 8,
		CircleSize:        4,
		HPDrainRate:       5,
		TimingPoints: []osu.TimingPoint{{
			BeatLength:  500,
			Meter:       4,
			Uninherited: true,
		}},
	}
	for i, tm := range times {
		bm.HitObjects = append(bm.HitObjects, osu.HitObject{
			Pos:  vector2d.Vector2dd{X: float64((i*71)%590 + 20), Y: float64((i*i*43+i*29)%430 + 20)},
			Time: tm,
			Type: osu.HitNormal,
		})
	}
	return bm
}

func curveTimes(points []StrainPoint) []int {
	times := make([]int, len(points))
	for i, p := range points {
		times[i] = p.Time
	}
	return times
}

func TestStaminaCurvePeakIsTheSkill(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		t.Run(mods.String(), func(t *testing.T) {
			curve := ProcessCurves(bm, mods, vars).Stamina
			skill := ProcessBeatmap(bm, mods, vars).Skills.Stamina

			if len(curve) == 0 {
				t.Fatal("empty stamina curve")
			}

			peak := math.Inf(-1)
			for _, p := range curve {
				peak = math.Max(peak, p.Value)
			}
			if peak != skill {
				t.Errorf("curve peak = %v, want the stamina skill %v", peak, skill)
			}
		})
	}
}

func TestStaminaCurveTimes(t *testing.T) {
	// 1220 is only 10 ms after 1210, so the strain skips it; 1500 follows 1220.
	bm := circlesAt(1000, 1100, 1210, 1220, 1500)
	vars := NewVars()

	got := curveTimes(ProcessCurves(bm, osu.NM, vars).Stamina)
	if want := []int{1100, 1210, 1500}; !reflect.DeepEqual(got, want) {
		t.Errorf("times = %v, want %v", got, want)
	}
}

func TestStaminaCurveWithSpeedMods(t *testing.T) {
	bm := circlesAt(1000, 1100, 1210, 1220, 1500)
	vars := NewVars()

	tests := []struct {
		mods  osu.Mod
		speed float64
	}{
		{osu.NM, 1},
		{osu.HD | osu.HR, 1},
		{osu.DT, 1.5},
		{osu.HT, 0.75},
	}
	for _, tc := range tests {
		t.Run(tc.mods.String(), func(t *testing.T) {
			curves := ProcessCurves(bm, tc.mods, vars)
			if curves.Speed != tc.speed {
				t.Errorf("Speed = %v, want %v", curves.Speed, tc.speed)
			}
			if curves.Mods != tc.mods || curves.ModsString != tc.mods.String() {
				t.Errorf("Mods = %v (%q), want %v (%q)", curves.Mods, curves.ModsString, tc.mods, tc.mods.String())
			}

			// Times are those of the calculation itself, which ApplyMods rounds up.
			var want []int
			for _, original := range []int{1100, 1210, 1500} {
				want = append(want, int(math.Ceil(float64(original)/tc.speed)))
			}
			if got := curveTimes(curves.Stamina); !reflect.DeepEqual(got, want) {
				t.Errorf("times = %v, want %v", got, want)
			}
		})
	}
}

func TestStaminaCurveIsOrderedAndFinite(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		curve := ProcessCurves(bm, mods, vars).Stamina
		for i, p := range curve {
			if math.IsNaN(p.Value) || math.IsInf(p.Value, 0) || p.Value < 0 {
				t.Fatalf("%s: point %d has value %v", mods, i, p.Value)
			}
			if i > 0 && p.Time < curve[i-1].Time {
				t.Fatalf("%s: point %d at %d precedes point %d at %d", mods, i, p.Time, i-1, curve[i-1].Time)
			}
		}
	}
}

func TestProcessCurvesWithoutHitObjects(t *testing.T) {
	curves := ProcessCurves(&osu.Beatmap{}, osu.DT, NewVars())

	if curves == nil || curves.Speed != 1.5 || len(curves.Stamina) != 0 {
		t.Errorf("got %+v, want an empty curve with Speed 1.5", curves)
	}
}

func TestProcessCurvesWithOnePress(t *testing.T) {
	// No interval between presses, so there is nothing to measure.
	curves := ProcessCurves(circlesAt(1000), osu.NM, NewVars())

	if len(curves.Stamina) != 0 {
		t.Errorf("stamina curve = %v, want empty", curves.Stamina)
	}
}

func TestProcessCurvesDoesNotChangeSkills(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		before := *ProcessBeatmap(bm, mods, vars)
		ProcessCurves(bm, mods, vars)
		if after := *ProcessBeatmap(bm, mods, vars); after != before {
			t.Errorf("%s: skills changed after calculating curves:\n got: %#v\nwant: %#v", mods, after, before)
		}
	}
}

func TestProcessCurvesConcurrentAndReadOnly(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	before := *bm
	before.TimingPoints = append([]osu.TimingPoint(nil), bm.TimingPoints...)
	before.HitObjects = make([]osu.HitObject, len(bm.HitObjects))
	for i := range bm.HitObjects {
		before.HitObjects[i] = bm.HitObjects[i].Clone()
	}

	mods := DefaultModCombinations()
	want := make([]*Curves, len(mods))
	for i, m := range mods {
		want[i] = ProcessCurves(bm, m, vars)
	}

	var wg sync.WaitGroup
	for run := 0; run < 4; run++ {
		for i, m := range mods {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if got := ProcessCurves(bm, m, vars); !reflect.DeepEqual(got, want[i]) {
					t.Errorf("%s: concurrent result differs from the sequential one", m)
				}
			}()
		}
	}
	wg.Wait()

	if !reflect.DeepEqual(before, *bm) {
		t.Fatal("ProcessCurves modified the source beatmap")
	}
}

func TestAgilityCurvePeakIsTheSkill(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		t.Run(mods.String(), func(t *testing.T) {
			curve := ProcessCurves(bm, mods, vars).Agility
			skill := ProcessBeatmap(bm, mods, vars).Skills.Agility

			if len(curve) == 0 {
				t.Fatal("empty agility curve")
			}

			peak := math.Inf(-1)
			for _, p := range curve {
				peak = math.Max(peak, p.Value)
			}
			if peak != skill {
				t.Errorf("curve peak = %v, want the agility skill %v", peak, skill)
			}
		})
	}
}

func TestAgilityCurveTimes(t *testing.T) {
	vars := NewVars()

	tests := []struct {
		name  string
		times []int
		want  []int
	}{
		{"every press", []int{1000, 1100, 1210, 1220, 1500}, []int{1000, 1100, 1210, 1220, 1500}},
		// The third note is at the same time as the second, so there is no
		// interval to measure and no strain is recorded for it.
		{"notes at the same time", []int{1000, 1100, 1100, 1300}, []int{1000, 1100, 1300}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := curveTimes(ProcessCurves(circlesAt(tc.times...), osu.NM, vars).Agility)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("times = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAgilityCurveWithSpeedMods(t *testing.T) {
	bm := circlesAt(1000, 1100, 1210, 1220, 1500)
	vars := NewVars()

	for _, mods := range []osu.Mod{osu.NM, osu.DT, osu.HT, osu.HD | osu.HR} {
		t.Run(mods.String(), func(t *testing.T) {
			speed := modsSpeed(mods)

			var want []int
			for _, original := range []int{1000, 1100, 1210, 1220, 1500} {
				want = append(want, int(math.Ceil(float64(original)/speed)))
			}
			if got := curveTimes(ProcessCurves(bm, mods, vars).Agility); !reflect.DeepEqual(got, want) {
				t.Errorf("times = %v, want %v", got, want)
			}
		})
	}
}

// The curve is the strains on another scale: it must not reverse their order,
// keep their zeros, and have a point for every strain.
func TestAgilityCurveKeepsTheShapeOfTheStrains(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		md := NewMapData(ApplyMods(bm, mods), mods)
		md.prepare()
		md.calculateAimStrains(vars)
		curve := md.agilityCurve(vars)

		if len(curve) != len(md.AimStrains) || len(md.AimStrainTimes) != len(md.AimStrains) {
			t.Fatalf("%s: %d points, %d times for %d strains", mods, len(curve), len(md.AimStrainTimes), len(md.AimStrains))
		}

		for i, strain := range md.AimStrains {
			if curve[i].Time != md.AimStrainTimes[i] {
				t.Fatalf("%s: point %d has time %d, want %d", mods, i, curve[i].Time, md.AimStrainTimes[i])
			}
			if (strain == 0) != (curve[i].Value == 0) {
				t.Fatalf("%s: point %d has strain %v and value %v", mods, i, strain, curve[i].Value)
			}
			if i == 0 {
				continue
			}

			// Strains that differ in the last bit may end up with the same
			// value, so the order may be lost but never reversed.
			prev := md.AimStrains[i-1]
			switch {
			case strain > prev && curve[i].Value < curve[i-1].Value,
				strain < prev && curve[i].Value > curve[i-1].Value,
				strain == prev && curve[i].Value != curve[i-1].Value:
				t.Fatalf("%s: strains %v -> %v became values %v -> %v", mods, prev, strain, curve[i-1].Value, curve[i].Value)
			}
		}
	}
}

// The scale of the curve is fixed by two things: its maximum is the skill, and
// the values follow the strains as a power law with the skill's exponent. This
// is checked without going through the code that builds the curve.
func TestAgilityCurveFollowsThePowerLawOfTheStrains(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		md := NewMapData(ApplyMods(bm, mods), mods)
		md.prepare()
		md.calculateAimStrains(vars)
		curve := md.agilityCurve(vars)

		maxStrain, peak := MaxFloatSliceValue(md.AimStrains), 0.0
		for _, p := range curve {
			peak = math.Max(peak, p.Value)
		}
		if maxStrain == 0 || peak == 0 {
			t.Fatalf("%s: maxStrain = %v, peak = %v", mods, maxStrain, peak)
		}

		for i, strain := range md.AimStrains {
			want := math.Pow(strain/maxStrain, vars.Agility.TotalPow)
			got := curve[i].Value / peak
			if math.Abs(got-want) > 1e-9*math.Max(1, want) {
				t.Fatalf("%s: point %d has %v of the peak, want %v (strain %v of %v)", mods, i, got, want, strain, maxStrain)
			}
		}
	}
}

func TestAgilityCurveIsOrderedAndFinite(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		curve := ProcessCurves(bm, mods, vars).Agility
		for i, p := range curve {
			if math.IsNaN(p.Value) || math.IsInf(p.Value, 0) || p.Value < 0 {
				t.Fatalf("%s: point %d has value %v", mods, i, p.Value)
			}
			if i > 0 && p.Time < curve[i-1].Time {
				t.Fatalf("%s: point %d at %d precedes point %d at %d", mods, i, p.Time, i-1, curve[i-1].Time)
			}
		}
	}
}

// Without a strain peak (here: two notes) the skill is 0, and so is the curve.
func TestAgilityCurveWithoutPeaks(t *testing.T) {
	curve := ProcessCurves(circlesAt(1000, 1100), osu.NM, NewVars()).Agility

	if len(curve) != 2 {
		t.Fatalf("curve = %v, want 2 points", curve)
	}
	for _, p := range curve {
		if p.Value != 0 {
			t.Errorf("curve = %v, want all zeros", curve)
		}
	}
}

func TestReadingCurvePeakIsTheSkill(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		t.Run(mods.String(), func(t *testing.T) {
			curve := ProcessCurves(bm, mods, vars).Reading
			skill := ProcessBeatmap(bm, mods, vars).Skills.Reading

			if len(curve) == 0 {
				t.Fatal("empty reading curve")
			}

			peak := math.Inf(-1)
			for _, p := range curve {
				peak = math.Max(peak, p.Value)
			}
			if peak != skill {
				t.Errorf("curve peak = %v, want the reading skill %v", peak, skill)
			}
		})
	}
}

// Every note has a reading point, so the curve has a point for each of them at
// its time (Ceil-rescaled by speed mods).
func TestReadingCurveTimes(t *testing.T) {
	vars := NewVars()
	times := []int{1000, 1100, 1100, 1210, 1500}

	for _, mods := range []osu.Mod{osu.NM, osu.DT, osu.HT, osu.HD | osu.HR} {
		t.Run(mods.String(), func(t *testing.T) {
			speed := modsSpeed(mods)
			var want []int
			for _, original := range times {
				want = append(want, int(math.Ceil(float64(original)/speed)))
			}
			if got := curveTimes(ProcessCurves(circlesAt(times...), mods, vars).Reading); !reflect.DeepEqual(got, want) {
				t.Errorf("times = %v, want %v", got, want)
			}
		})
	}
}

func TestReadingCurveFollowsThePowerLawOfTheStrains(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		md := NewMapData(ApplyMods(bm, mods), mods)
		md.prepare()
		md.calculateReading(vars, md.HasMod(osu.HD))
		curve := md.readingCurve(vars)

		maxStrain, peak := MaxFloatSliceValue(md.ReadingStrains), 0.0
		for _, p := range curve {
			peak = math.Max(peak, p.Value)
		}
		if maxStrain == 0 || peak == 0 {
			t.Fatalf("%s: maxStrain = %v, peak = %v", mods, maxStrain, peak)
		}

		for i, strain := range md.ReadingStrains {
			want := math.Pow(strain/maxStrain, vars.Reading.TotalPow)
			got := curve[i].Value / peak
			if math.Abs(got-want) > 1e-9*math.Max(1, want) {
				t.Fatalf("%s: point %d has %v of the peak, want %v", mods, i, got, want)
			}
		}
	}
}

func TestReadingCurveIsOrderedAndFinite(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		curve := ProcessCurves(bm, mods, vars).Reading
		for i, p := range curve {
			if math.IsNaN(p.Value) || math.IsInf(p.Value, 0) || p.Value < 0 {
				t.Fatalf("%s: point %d has value %v", mods, i, p.Value)
			}
			if i > 0 && p.Time < curve[i-1].Time {
				t.Fatalf("%s: point %d at %d precedes point %d at %d", mods, i, p.Time, i-1, curve[i-1].Time)
			}
		}
	}
}

// A map too short to have a strain peak has a reading of 0, and so has its curve.
func TestReadingCurveWithoutPeaks(t *testing.T) {
	curve := ProcessCurves(circlesAt(1000, 1100), osu.NM, NewVars()).Reading

	if len(curve) != 2 {
		t.Fatalf("curve = %v, want 2 points", curve)
	}
	for _, p := range curve {
		if p.Value != 0 {
			t.Errorf("curve = %v, want all zeros", curve)
		}
	}
}

func TestReactionCurvePeakIsTheSkill(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		t.Run(mods.String(), func(t *testing.T) {
			curve := ProcessCurves(bm, mods, vars).Reaction
			skill := ProcessBeatmap(bm, mods, vars).Skills.Reaction

			if len(curve) == 0 {
				t.Fatal("empty reaction curve")
			}

			peak := math.Inf(-1)
			for _, p := range curve {
				peak = math.Max(peak, p.Value)
			}
			if peak != skill {
				t.Errorf("curve peak = %v, want the reaction skill %v", peak, skill)
			}
		})
	}
}

// One point per target point: a circle's start or a slider tick.
func TestReactionCurveTimes(t *testing.T) {
	vars := NewVars()
	times := []int{1000, 1100, 1210, 1500, 1503, 2000}

	for _, mods := range []osu.Mod{osu.NM, osu.DT, osu.HT} {
		t.Run(mods.String(), func(t *testing.T) {
			speed := modsSpeed(mods)
			// The note 3 ms after another is dropped by the 5 ms filter.
			var want []int
			for _, original := range []int{1000, 1100, 1210, 1500, 2000} {
				want = append(want, int(math.Ceil(float64(original)/speed)))
			}
			if got := curveTimes(ProcessCurves(circlesAt(times...), mods, vars).Reaction); !reflect.DeepEqual(got, want) {
				t.Errorf("times = %v, want %v", got, want)
			}
		})
	}
}

// The curve is proportional to the per-point reaction values.
func TestReactionCurveIsProportionalToTheValues(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		md := NewMapData(ApplyMods(bm, mods), mods)
		md.prepare()
		values := md.reactionValues(vars, md.HasMod(osu.HD))
		curve := md.reactionCurve(vars, md.HasMod(osu.HD))

		if len(curve) != len(values) {
			t.Fatalf("%s: %d points for %d values", mods, len(curve), len(values))
		}
		maxValue, peak := MaxFloatSliceValue(values), 0.0
		for _, p := range curve {
			peak = math.Max(peak, p.Value)
		}
		if maxValue == 0 || peak == 0 {
			t.Fatalf("%s: max = %v, peak = %v", mods, maxValue, peak)
		}
		for i, v := range values {
			want, got := v/maxValue, curve[i].Value/peak
			if math.Abs(got-want) > 1e-9 {
				t.Fatalf("%s: point %d has %v of the peak, want %v", mods, i, got, want)
			}
		}
	}
}

func TestReactionCurveIsOrderedAndFinite(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		curve := ProcessCurves(bm, mods, vars).Reaction
		for i, p := range curve {
			if math.IsNaN(p.Value) || math.IsInf(p.Value, 0) || p.Value < 0 {
				t.Fatalf("%s: point %d has value %v", mods, i, p.Value)
			}
			if i > 0 && p.Time < curve[i-1].Time {
				t.Fatalf("%s: point %d at %d precedes point %d at %d", mods, i, p.Time, i-1, curve[i-1].Time)
			}
		}
	}
}

// reactionSkill must give what the skill always was; pin it on hand-made values.
func TestReactionSkillOfValues(t *testing.T) {
	vars := NewVars()
	w := vars.Reaction.AvgWeighting

	if got := reactionSkill(nil, vars); got != 0 {
		t.Errorf("no values: %v, want 0", got)
	}
	// 10 sets avg to w*10; 4 is below half of the max (10) and is ignored;
	// 8 moves the average on.
	avg := w * 10
	avg = w*8 + (1-w)*avg
	if got, want := reactionSkill([]float64{10, 4, 8}, vars), (10+avg)/2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPrecisionAndAccuracyCurvePeakIsTheSkill(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		t.Run(mods.String(), func(t *testing.T) {
			curves := ProcessCurves(bm, mods, vars)
			skills := ProcessBeatmap(bm, mods, vars).Skills

			for _, tc := range []struct {
				name  string
				curve []StrainPoint
				skill float64
			}{
				{"precision", curves.Precision, skills.Precision},
				{"accuracy", curves.Accuracy, skills.Accuracy},
			} {
				if len(tc.curve) == 0 {
					t.Fatalf("empty %s curve", tc.name)
				}
				peak := math.Inf(-1)
				for _, p := range tc.curve {
					peak = math.Max(peak, p.Value)
				}
				if peak != tc.skill {
					t.Errorf("%s curve peak = %v, want the skill %v", tc.name, peak, tc.skill)
				}
			}
		})
	}
}

// The curves are the agility and stamina curves put through the skills'
// formulas: same times and the same order of values.
func TestPrecisionAndAccuracyCurveFollowTheirSource(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		c := ProcessCurves(bm, mods, vars)

		for _, tc := range []struct {
			name     string
			src, got []StrainPoint
		}{
			{"precision", c.Agility, c.Precision},
			{"accuracy", c.Stamina, c.Accuracy},
		} {
			if !reflect.DeepEqual(curveTimes(tc.src), curveTimes(tc.got)) {
				t.Fatalf("%s %s: times differ from the source curve", mods, tc.name)
			}
			for i := range tc.src {
				if math.IsNaN(tc.got[i].Value) || math.IsInf(tc.got[i].Value, 0) || tc.got[i].Value < 0 {
					t.Fatalf("%s %s: point %d has value %v", mods, tc.name, i, tc.got[i].Value)
				}
				for j := range tc.src {
					if tc.src[i].Value < tc.src[j].Value && tc.got[i].Value > tc.got[j].Value {
						t.Fatalf("%s %s: points %d and %d are in the opposite order of the source", mods, tc.name, i, j)
					}
				}
			}
		}
	}
}

// Precision and accuracy of a single value, with the numbers worked out by hand.
func TestPrecisionAndAccuracyFormulas(t *testing.T) {
	vars := NewVars()

	// precision: 20 * ((1+a)^0.1 - 0.995462)*cs, squared. a = 0, cs = 4.
	scaled := (1 - vars.Precision.AgilitySubtract) * 4
	if got, want := precisionFromAgility(0, 4, vars), vars.Precision.TotalMult*scaled*scaled; math.Abs(got-want) > 1e-12 {
		t.Errorf("precision = %v, want %v", got, want)
	}

	// accuracy: 23100 * (s^0.3 / ssUr)^1.3. s = 1, ssUr = 2.
	want := vars.Accuracy.TotalMult * math.Pow(0.5, vars.Accuracy.TotalPow)
	if got := accuracyFromStamina(1, 2, vars); math.Abs(got-want) > 1e-9 {
		t.Errorf("accuracy = %v, want %v", got, want)
	}
}

// Maps without circles have no SS_UR; accuracy is 0, not NaN.
func TestAccuracyCurveWithoutCircles(t *testing.T) {
	bm := curvesTestBeatmap(t)
	for i := range bm.HitObjects {
		bm.HitObjects[i].Type = osu.HitSlider
	}
	curves := ProcessCurves(bm, osu.NM, NewVars())
	for _, p := range curves.Accuracy {
		if math.IsNaN(p.Value) {
			t.Fatalf("accuracy curve has NaN: %v", curves.Accuracy)
		}
	}
}

// Memory only exists with FL; the curve ends at the skill and never falls.
func TestMemoryCurveEndsAtTheSkill(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	withFL := 0
	for _, mods := range DefaultModCombinations() {
		curve := ProcessCurves(bm, mods, vars).Memory

		if mods&osu.FL == 0 {
			if len(curve) != 0 {
				t.Errorf("%s: memory curve without FL: %v", mods, curve)
			}
			continue
		}
		withFL++

		skill := ProcessBeatmap(bm, mods, vars).Skills.Memory
		if skill == 0 || len(curve) == 0 {
			t.Fatalf("%s: skill = %v, %d points", mods, skill, len(curve))
		}
		if last := curve[len(curve)-1].Value; last != skill {
			t.Errorf("%s: curve ends at %v, want the memory skill %v", mods, last, skill)
		}
		for i, p := range curve {
			if math.IsNaN(p.Value) || math.IsInf(p.Value, 0) || p.Value < 0 {
				t.Fatalf("%s: point %d has value %v", mods, i, p.Value)
			}
			if i > 0 && (p.Time < curve[i-1].Time || p.Value < curve[i-1].Value) {
				t.Fatalf("%s: point %d (%d, %v) falls or goes back in time after (%d, %v)", mods, i, p.Time, p.Value, curve[i-1].Time, curve[i-1].Value)
			}
		}
	}
	if withFL == 0 {
		t.Fatal("no mod combination with FL")
	}
}

// Spinners and the first object have no point; every other object has one.
func TestMemoryCurveTimes(t *testing.T) {
	bm := circlesAt(1000, 1500, 2000, 2500)
	bm.HitObjects[2].Type = osu.HitSpinner
	vars := NewVars()

	for _, mods := range []osu.Mod{osu.FL, osu.FL | osu.DT, osu.FL | osu.HT} {
		speed := modsSpeed(mods)
		var want []int
		for _, original := range []int{1500, 2500} {
			want = append(want, int(math.Ceil(float64(original)/speed)))
		}
		if got := curveTimes(ProcessCurves(bm, mods, vars).Memory); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: times = %v, want %v", mods, got, want)
		}
	}
}

func TestMemoryCurveIsTheScaleOfTheRunningTotal(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()
	md := NewMapData(ApplyMods(bm, osu.FL), osu.FL)
	md.prepare()
	md.calculateMemory(vars)
	curve := md.memoryCurve(vars)

	if len(curve) != len(md.MemoryTotals) {
		t.Fatalf("%d points for %d totals", len(curve), len(md.MemoryTotals))
	}
	for i, total := range md.MemoryTotals {
		want := vars.Memory.TotalMult * math.Pow(total, vars.Memory.TotalPow)
		if curve[i].Value != want {
			t.Fatalf("point %d = %v, want %v", i, curve[i].Value, want)
		}
	}
	if md.MemoryTotals[len(md.MemoryTotals)-1] == 0 {
		t.Fatal("the test map has no memory points")
	}
}

// One object: no memory points, no curve, no panic.
func TestMemoryCurveWithOneObject(t *testing.T) {
	curves := ProcessCurves(circlesAt(1000), osu.FL, NewVars())
	if len(curves.Memory) != 0 {
		t.Errorf("curve = %v, want empty", curves.Memory)
	}
}

// streamsAt builds a map of streams: each has count presses `interval` ms apart,
// starting at the given times.
func streamsAt(interval, count int, starts ...int) []int {
	var times []int
	for _, start := range starts {
		for k := 0; k < count; k++ {
			times = append(times, start+interval*k)
		}
	}
	return times
}

func TestTenacityCurveIsAStepCurveOfTheStreams(t *testing.T) {
	vars := NewVars()
	times := append(streamsAt(100, 10, 1000), streamsAt(80, 8, 5000)...)
	curves := ProcessCurves(circlesAt(times...), osu.NM, vars)

	// The streams (7+ presses at one interval) the map has: the last press of
	// each is not counted in it by the skill's own stream detection.
	slow := tenacityOfStream(100, 10, vars)
	fast := tenacityOfStream(80, 8, vars)
	want := []StrainPoint{
		{999, 0}, {1000, slow}, {1800, slow}, {1801, 0},
		{4999, 0}, {5000, fast}, {5480, fast}, {5481, 0},
	}
	if !reflect.DeepEqual(curves.Tenacity, want) {
		t.Errorf("curve = %v\nwant    %v", curves.Tenacity, want)
	}
	// The skill comes from the fastest stream.
	if skill := ProcessBeatmap(circlesAt(times...), osu.NM, vars).Skills.Tenacity; skill != fast {
		t.Errorf("skill = %v, want %v", skill, fast)
	}
}

func TestTenacityCurvePeakIsTheSkill(t *testing.T) {
	bm := curvesTestBeatmap(t)
	vars := NewVars()

	for _, mods := range DefaultModCombinations() {
		curve := ProcessCurves(bm, mods, vars).Tenacity
		skill := ProcessBeatmap(bm, mods, vars).Skills.Tenacity

		if skill == 0 || len(curve) == 0 {
			t.Fatalf("%s: skill = %v, %d points", mods, skill, len(curve))
		}
		peak := math.Inf(-1)
		for i, p := range curve {
			peak = math.Max(peak, p.Value)
			if math.IsNaN(p.Value) || math.IsInf(p.Value, 0) || p.Value < 0 {
				t.Fatalf("%s: point %d has value %v", mods, i, p.Value)
			}
			if i > 0 && p.Time < curve[i-1].Time {
				t.Fatalf("%s: point %d at %d precedes point %d at %d", mods, i, p.Time, i-1, curve[i-1].Time)
			}
		}
		if peak != skill {
			t.Errorf("%s: curve peak = %v, want the tenacity skill %v", mods, peak, skill)
		}
	}
}

// A slow but very long stream could be worth more than the fast stream the
// skill is taken from; the curve must not go above the skill.
func TestTenacityCurveNeverExceedsTheSkill(t *testing.T) {
	vars := NewVars()
	if tenacityOfStream(100, 80, vars) <= tenacityOfStream(80, 8, vars) {
		t.Skip("the default variables do not make a slow stream worth more")
	}
	times := append(streamsAt(100, 80, 1000), streamsAt(80, 8, 20000)...)
	curves := ProcessCurves(circlesAt(times...), osu.NM, vars)
	skill := ProcessBeatmap(circlesAt(times...), osu.NM, vars).Skills.Tenacity

	peak, atSlow := 0.0, 0.0
	for _, p := range curves.Tenacity {
		peak = math.Max(peak, p.Value)
		if p.Time == 1000 {
			atSlow = p.Value
		}
	}
	if peak != skill || atSlow != skill {
		t.Errorf("peak = %v, slow stream = %v, want both the skill %v", peak, atSlow, skill)
	}
}

// Streams are only counted from 7 presses; a map without any has no curve.
func TestTenacityCurveWithoutStreams(t *testing.T) {
	curves := ProcessCurves(circlesAt(1000, 1100, 1200, 3000, 3100), osu.NM, NewVars())
	if len(curves.Tenacity) != 0 {
		t.Errorf("curve = %v, want empty", curves.Tenacity)
	}
}

func TestTenacityCurveWithSpeedMods(t *testing.T) {
	vars := NewVars()
	times := streamsAt(100, 10, 1000)

	for _, mods := range []osu.Mod{osu.DT, osu.HT} {
		curve := ProcessCurves(circlesAt(times...), mods, vars).Tenacity
		speed := modsSpeed(mods)
		start := int(math.Ceil(1000 / speed))
		if len(curve) != 4 || curve[1].Time != start {
			t.Errorf("%s: curve = %v, want a plateau starting at %d", mods, curve, start)
		}
	}
}
