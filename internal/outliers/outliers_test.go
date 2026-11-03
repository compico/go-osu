package outliers

import (
	"math"
	"testing"
)

// rowsOnCurve builds beatmaps exactly on skill = c * stars^p.
func rowsOnCurve(c, p float64, stars ...float64) []Row {
	var rows []Row
	for i, s := range stars {
		rows = append(rows, Row{BeatmapID: int32(i + 1), Stars: s, Skill: c * math.Pow(s, p)})
	}
	return rows
}

func TestFitRecoversThePowerLaw(t *testing.T) {
	rows := rowsOnCurve(50, 1.3, 2, 3, 4.5, 6, 7.5, 9)
	m, err := Fit(rows)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(m.B-1.3) > 1e-9 || math.Abs(math.Exp(m.A)-50) > 1e-6 {
		t.Errorf("fit = %+v, want 50*stars^1.3", m)
	}
	if m.Sigma > 1e-9 || m.N != 6 {
		t.Errorf("sigma = %v, n = %d", m.Sigma, m.N)
	}
	if got, want := m.Expected(5), 50*math.Pow(5, 1.3); math.Abs(got-want) > 1e-6 {
		t.Errorf("expected(5) = %v, want %v", got, want)
	}
}

func TestRankPutsTheOutlierFirst(t *testing.T) {
	rows := rowsOnCurve(50, 1.3, 2, 3, 4, 5, 6, 7, 8, 9)
	rows[2].Skill *= 3 // 4 stars, 3 times what it should be
	rows[5].Skill /= 2 // 7 stars, half of it
	m, err := Fit(rows)
	if err != nil {
		t.Fatal(err)
	}
	ranked := Rank(rows, m)
	if ranked[0].BeatmapID != rows[2].BeatmapID {
		t.Errorf("first = %d, want %d", ranked[0].BeatmapID, rows[2].BeatmapID)
	}
	if last := ranked[len(ranked)-1]; last.BeatmapID != rows[5].BeatmapID {
		t.Errorf("last = %d, want %d", last.BeatmapID, rows[5].BeatmapID)
	}
	if ranked[0].Ratio < 2 || ranked[0].Z <= 0 {
		t.Errorf("outlier = %+v", ranked[0])
	}
	if math.Abs(ranked[0].Residual-math.Log(ranked[0].Ratio)) > 1e-12 {
		t.Errorf("residual %v is not ln(ratio %v)", ranked[0].Residual, ranked[0].Ratio)
	}
	for i := 1; i < len(ranked); i++ {
		if ranked[i].Residual > ranked[i-1].Residual {
			t.Fatalf("not sorted at %d", i)
		}
	}
}

func TestFitIgnoresRowsThatCannotBeLogged(t *testing.T) {
	rows := rowsOnCurve(50, 1.3, 2, 3, 4, 5)
	rows = append(rows,
		Row{BeatmapID: 90, Stars: 0, Skill: 10},
		Row{BeatmapID: 91, Stars: 5, Skill: 0},
		Row{BeatmapID: 92, Stars: math.NaN(), Skill: 10},
		Row{BeatmapID: 93, Stars: 5, Skill: math.Inf(1)},
	)
	m, err := Fit(rows)
	if err != nil {
		t.Fatal(err)
	}
	if m.N != 4 {
		t.Errorf("n = %d, want 4", m.N)
	}
	if got := len(Rank(rows, m)); got != 4 {
		t.Errorf("ranked %d rows, want 4", got)
	}
}

func TestFitErrors(t *testing.T) {
	if _, err := Fit(nil); err == nil {
		t.Error("no rows: want an error")
	}
	if _, err := Fit(rowsOnCurve(50, 1, 2, 3)); err == nil {
		t.Error("two rows: want an error")
	}
	if _, err := Fit(rowsOnCurve(50, 1, 4, 4, 4)); err == nil {
		t.Error("one star rating: want an error")
	}
}

func TestRankBreaksTiesByID(t *testing.T) {
	rows := []Row{{BeatmapID: 3, Stars: 4, Skill: 100}, {BeatmapID: 2, Stars: 4, Skill: 100}, {BeatmapID: 1, Stars: 5, Skill: 200}, {BeatmapID: 4, Stars: 6, Skill: 300}}
	m, err := Fit(rows)
	if err != nil {
		t.Fatal(err)
	}
	ranked := Rank(rows, m)
	var ids []int32
	for _, r := range ranked {
		if r.Stars == 4 {
			ids = append(ids, r.BeatmapID)
		}
	}
	if len(ids) != 2 || ids[0] != 2 || ids[1] != 3 {
		t.Errorf("tie order = %v, want [2 3]", ids)
	}
}
