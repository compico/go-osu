// Package outliers finds beatmaps whose skill is far from what their star
// rating predicts.
//
// The skill is fitted to the stars on a log-log scale, ln(skill) = a + b*ln(stars),
// over all given beatmaps. A beatmap's residual is how far ln(skill) is above
// the fit: a ratio of exp(residual) between the real skill and the expected one.
package outliers

import (
	"errors"
	"math"
	"sort"
)

// Row is one beatmap's star rating and skill.
type Row struct {
	BeatmapID  int32
	Title      string
	Artist     string
	Difficulty string
	Stars      float64
	Skill      float64
}

// Model is the fit ln(skill) = A + B*ln(stars).
type Model struct {
	A, B float64
	// Sigma is the standard deviation of the residuals.
	Sigma float64
	// N is the number of rows the model was fitted to.
	N int
}

// Expected returns the skill the model expects at the given stars.
func (m Model) Expected(stars float64) float64 {
	return math.Exp(m.A + m.B*math.Log(stars))
}

// Ranked is a Row together with how far it is from the model.
type Ranked struct {
	Row
	Expected float64
	// Ratio is Skill / Expected.
	Ratio float64
	// Residual is ln(Ratio).
	Residual float64
	// Z is Residual in standard deviations.
	Z float64
}

// usable reports whether a row can be put on a log scale.
func usable(r Row) bool {
	return r.Stars > 0 && r.Skill > 0 &&
		!math.IsNaN(r.Stars) && !math.IsInf(r.Stars, 0) &&
		!math.IsNaN(r.Skill) && !math.IsInf(r.Skill, 0)
}

// Fit fits the model by least squares. Rows with a non-positive or non-finite
// star rating or skill are ignored. It needs at least 3 usable rows with
// different star ratings.
func Fit(rows []Row) (Model, error) {
	var xs, ys []float64
	for _, r := range rows {
		if usable(r) {
			xs = append(xs, math.Log(r.Stars))
			ys = append(ys, math.Log(r.Skill))
		}
	}
	n := float64(len(xs))
	if len(xs) < 3 {
		return Model{}, errors.New("outliers: need at least 3 beatmaps with a positive star rating and skill")
	}

	var mx, my float64
	for i := range xs {
		mx += xs[i]
		my += ys[i]
	}
	mx /= n
	my /= n

	var sxx, sxy float64
	for i := range xs {
		sxx += (xs[i] - mx) * (xs[i] - mx)
		sxy += (xs[i] - mx) * (ys[i] - my)
	}
	if sxx == 0 {
		return Model{}, errors.New("outliers: all beatmaps have the same star rating")
	}

	b := sxy / sxx
	a := my - b*mx

	var ss float64
	for i := range xs {
		e := ys[i] - (a + b*xs[i])
		ss += e * e
	}
	return Model{A: a, B: b, Sigma: math.Sqrt(ss / n), N: len(xs)}, nil
}

// Rank returns the usable rows ordered by how far above the model they are,
// the highest first.
func Rank(rows []Row, m Model) []Ranked {
	out := make([]Ranked, 0, len(rows))
	for _, r := range rows {
		if !usable(r) {
			continue
		}
		expected := m.Expected(r.Stars)
		resid := math.Log(r.Skill / expected)
		z := 0.0
		if m.Sigma > 0 {
			z = resid / m.Sigma
		}
		out = append(out, Ranked{Row: r, Expected: expected, Ratio: r.Skill / expected, Residual: resid, Z: z})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Residual != out[j].Residual {
			return out[i].Residual > out[j].Residual
		}
		return out[i].BeatmapID < out[j].BeatmapID
	})
	return out
}
