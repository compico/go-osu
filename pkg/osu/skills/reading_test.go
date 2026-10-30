package skills

import (
	"math"
	"testing"
)

func TestReadingCosineApproxErrorBound(t *testing.T) {
	const maxError = 3.2e-6
	fractions := [...]float64{0.25, 0.5, 0.75}
	maxObservedError := 0.0

	for i := 0; i < readingCosineTableSize-1; i++ {
		for _, fraction := range fractions {
			x := (float64(i) + fraction) * readingCosineTableStep
			error := math.Abs(readingCosineApprox(x) - math.Cos(x))
			if error > maxObservedError {
				maxObservedError = error
			}
		}
	}

	if maxObservedError > maxError {
		t.Fatalf("maximum cosine error %g exceeds bound %g", maxObservedError, maxError)
	}
}

func TestReadingCosineApproxBoundaries(t *testing.T) {
	for _, x := range []float64{0, readingCosineTableMax} {
		if got, want := readingCosineApprox(x), math.Cos(x); got != want {
			t.Errorf("readingCosine(%g) = %g, want %g", x, got, want)
		}
	}

	if !math.IsNaN(readingCosineApprox(math.NaN())) {
		t.Error("readingCosineApprox(NaN) must return NaN")
	}
}
