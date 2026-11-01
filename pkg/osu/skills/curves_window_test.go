package skills

import (
	"reflect"
	"testing"
)

func TestMaxPerWindow(t *testing.T) {
	curve := []StrainPoint{
		{0, 1}, {100, 5}, {399, 3}, // window 0: the peak is 5 at 100
		{400, 2}, {500, 2}, // window 1: equal points, the first wins
		{1300, 7},                       // windows 2 and 3 are empty (a break)
		{1700, 1}, {1799, 4}, {1800, 9}, // window 4 ends at 1999: 1700, 1799, 1800 are in 4
	}
	tests := []struct {
		name   string
		window int
		want   []StrainPoint
	}{
		{"400", 400, []StrainPoint{{100, 5}, {400, 2}, {1300, 7}, {1800, 9}}},
		{"one window", 100000, []StrainPoint{{1800, 9}}},
		{"window of one point", 1, curve},
		{"zero is the raw curve", 0, curve},
		{"negative is the raw curve", -5, curve},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := MaxPerWindow(curve, tc.window); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got  %v\nwant %v", got, tc.want)
			}
		})
	}
}

func TestMaxPerWindowKeepsTheHighestPointAndOrder(t *testing.T) {
	bm := curvesTestBeatmap(t)
	curves := ProcessCurves(bm, 0, NewVars())

	for _, curve := range [][]StrainPoint{curves.Stamina, curves.Agility, curves.Reading, curves.Reaction} {
		top := 0.0
		for _, p := range curve {
			top = max(top, p.Value)
		}
		for _, window := range []int{1, 50, 400, 3000} {
			got := MaxPerWindow(curve, window)
			if len(got) > len(curve) || len(got) == 0 {
				t.Fatalf("window %d: %d points from %d", window, len(got), len(curve))
			}
			peak := 0.0
			for i, p := range got {
				peak = max(peak, p.Value)
				if i > 0 && p.Time < got[i-1].Time {
					t.Fatalf("window %d: not ordered", window)
				}
			}
			if peak != top {
				t.Errorf("window %d: peak %v, want %v", window, peak, top)
			}
		}
	}
}

func TestMaxPerWindowDoesNotChangeItsInputAndHandlesEmpty(t *testing.T) {
	in := []StrainPoint{{10, 1}, {20, 3}, {30, 2}}
	MaxPerWindow(in, 100)
	if !reflect.DeepEqual(in, []StrainPoint{{10, 1}, {20, 3}, {30, 2}}) {
		t.Errorf("input changed: %v", in)
	}
	if got := MaxPerWindow(nil, 400); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

// Negative times (objects before 0) fall into windows of the same length.
func TestMaxPerWindowNegativeTimes(t *testing.T) {
	in := []StrainPoint{{-500, 1}, {-401, 4}, {-399, 2}, {-1, 3}, {0, 1}}
	// Windows of 400: [-800,-401] holds -500 and -401, [-400,-1] holds -399 and
	// -1, [0,399] holds 0.
	want := []StrainPoint{{-401, 4}, {-1, 3}, {0, 1}}
	if got := MaxPerWindow(in, 400); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
