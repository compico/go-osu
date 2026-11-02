package skills

import (
	"math"
	"reflect"
	"testing"
)

func TestResampleMax(t *testing.T) {
	curve := []StrainPoint{
		{0, 1}, {100, 5}, {399, 3}, // window 0: the strongest is 5
		{400, 2}, {500, 2}, // window 1
		{1300, 7},            // window 3; window 2 is empty
		{1700, 1}, {1799, 4}, // window 4
	}
	tests := []struct {
		name    string
		windows int
		carry   bool
		want    []float64
	}{
		{"empty windows are 0", 5, false, []float64{5, 2, 0, 7, 4}},
		{"carry keeps the last value", 5, true, []float64{5, 2, 2, 7, 4}},
		{"points past the grid fall into the last window", 3, false, []float64{5, 2, 7}},
		{"the grid can be longer than the curve", 7, false, []float64{5, 2, 0, 7, 4, 0, 0}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResampleMax(curve, 400, 1, tc.windows, tc.carry)
			var values []float64
			for i, p := range got {
				values = append(values, p.Value)
				if want := 400*i + 200; p.Time != want {
					t.Errorf("window %d: time %d, want the middle %d", i, p.Time, want)
				}
			}
			if !reflect.DeepEqual(values, tc.want) {
				t.Errorf("values %v, want %v", values, tc.want)
			}
		})
	}
}

// Windows are on the beatmap's timeline: with DT (speed 1.5) a window of 3000 ms
// holds 2000 ms of the curve's own time, and the times are given back on the
// curve's timeline.
func TestResampleMaxWithSpeed(t *testing.T) {
	curve := []StrainPoint{{0, 1}, {1999, 4}, {2000, 2}, {5000, 9}}
	got := ResampleMax(curve, 3000, 1.5, 3, false)

	want := []StrainPoint{{1000, 4}, {3000, 2}, {5000, 9}}
	// 5000 * 1.5 = 7500 is in window 2 [6000, 9000); 2000 * 1.5 = 3000 starts window 1.
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestResampleMaxRawAndEdgeCases(t *testing.T) {
	in := []StrainPoint{{10, 1}, {20, 3}, {30, 2}}

	raw := ResampleMax(in, 0, 1, 5, false)
	if !reflect.DeepEqual(raw, in) {
		t.Errorf("window 0: %v, want the raw curve", raw)
	}
	raw[0].Value = 99
	if in[0].Value != 1 {
		t.Error("window 0 returned the input itself, not a copy")
	}

	if got := ResampleMax(in, 100, 1, 0, false); len(got) != 0 || got == nil {
		t.Errorf("no windows: %v, want an empty, non-nil list", got)
	}
	if got := ResampleMax(nil, 100, 1, 2, true); !reflect.DeepEqual(got, []StrainPoint{{50, 0}, {150, 0}}) {
		t.Errorf("empty curve: %v, want zeros", got)
	}
	// A bad speed is taken as 1.
	if got := ResampleMax(in, 100, 0, 1, false); got[0].Time != 50 {
		t.Errorf("speed 0: %v", got)
	}
	// Negative times (objects before the start) fall into the first window.
	if got := ResampleMax([]StrainPoint{{-500, 4}, {50, 1}}, 100, 1, 2, false); got[0].Value != 4 {
		t.Errorf("negative time: %v", got)
	}
}

func TestWindowCount(t *testing.T) {
	a := []StrainPoint{{0, 1}, {9999, 1}}
	b := []StrainPoint{{0, 1}, {12000, 1}}
	if got := WindowCount(5000, 1, a, b, nil); got != 3 {
		t.Errorf("got %d, want 3 (12000 ms is in the third window)", got)
	}
	if got := WindowCount(5000, 1, a); got != 2 {
		t.Errorf("got %d, want 2 (9999 ms is in the second window)", got)
	}
	if got := WindowCount(5000, 2, a); got != 4 { // 9999 * 2 = 19998 -> window 3
		t.Errorf("got %d, want 4", got)
	}
	if got := WindowCount(5000, 1); got != 1 {
		t.Errorf("no curves: %d, want 1", got)
	}
	if got := WindowCount(0, 1, a); got != 0 {
		t.Errorf("window 0: %d, want 0", got)
	}
}

// A real curve keeps its highest point and gets a uniform, ordered grid.
func TestResampleMaxKeepsThePeakOfRealCurves(t *testing.T) {
	bm := curvesTestBeatmap(t)
	curves := ProcessCurves(bm, 0, NewVars())
	all := [][]StrainPoint{curves.Stamina, curves.Agility, curves.Reading, curves.Reaction, curves.Accuracy}

	for _, window := range []int{400, 5000, 10000} {
		n := WindowCount(window, curves.Speed, all...)
		for _, curve := range all {
			top := 0.0
			for _, p := range curve {
				top = math.Max(top, p.Value)
			}
			got := ResampleMax(curve, window, curves.Speed, n, false)
			if len(got) != n {
				t.Fatalf("window %d: %d points, want %d", window, len(got), n)
			}
			peak := 0.0
			for i, p := range got {
				peak = math.Max(peak, p.Value)
				if i > 0 && p.Time <= got[i-1].Time {
					t.Fatalf("window %d: not strictly increasing in time", window)
				}
			}
			if peak != top {
				t.Errorf("window %d: peak %v, want the curve's %v", window, peak, top)
			}
		}
	}
}
