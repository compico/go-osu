package skills

import (
	"math"
	"os"
	"sync"
	"testing"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/beatmap"
	"github.com/compico/go-osu/pkg/vector2d"
)

// buildFakeMapData constructs a MapData with hand-made ReadingPoints so we
// can exercise CalculateReading without the not-yet-ported aim/reading
// point preparation pipeline (GatherAimAndReadingPoints / CalculateAngles).
func buildFakeMapData(ar float64, n int, spacingMs int) *MapData {
	bm := &osu.Beatmap{ApproachRate: ar}
	md := NewMapData(bm, 0)

	preempt := int(arToMs(ar))
	x, y := 100.0, 100.0
	for i := 0; i < n; i++ {
		t := i * spacingMs
		x += 40
		if i%4 == 0 {
			y += 30
			x -= 20
		}
		pos := vector2d.Vector2dd{X: x, Y: y}

		var dist float64
		if i > 0 {
			prev := md.ReadingPoints[i-1].Pos
			dist = pos.DistanceFrom(prev)
		}

		md.ReadingPoints = append(md.ReadingPoints, ReadingPoint{
			Index:   i,
			Time:    t,
			Preempt: t - preempt,
			FadeIn:  t - preempt,
			FadeOut: t,
			Angle:   0,
			Pos:     pos,
			Dist:    dist,
		})
	}

	return md
}

func TestCalculateReading_NoPanicAndFinite(t *testing.T) {
	vars := DefaultVars()

	for _, n := range []int{0, 1, 2, 5, 50} {
		for _, hidden := range []bool{false, true} {
			md := buildFakeMapData(9.0, n, 150)
			CalculateReading(md, vars, hidden)

			if len(md.ReadingStrains) != n {
				t.Fatalf("n=%d hidden=%v: got %d strains, want %d", n, hidden, len(md.ReadingStrains), n)
			}

			if math.IsNaN(md.Skills.Reading) || math.IsInf(md.Skills.Reading, 0) {
				t.Fatalf("n=%d hidden=%v: reading skill is not finite: %v", n, hidden, md.Skills.Reading)
			}

			if md.Skills.Reading < 0 {
				t.Fatalf("n=%d hidden=%v: reading skill is negative: %v", n, hidden, md.Skills.Reading)
			}
		}
	}
}

// TestCalculateReading_Concurrent exercises many independent MapData values
// being calculated concurrently, which only makes sense (and is only race-
// free) if no shared mutable state leaks between them. Run with -race.
func TestCalculateReading_Concurrent(t *testing.T) {
	vars := DefaultVars() // one immutable *Vars, shared read-only by every goroutine

	var wg sync.WaitGroup
	for g := 0; g < 50; g++ {
		g := g
		wg.Add(1)
		go func() {
			defer wg.Done()
			md := buildFakeMapData(8.0+float64(g%3), 40, 120) // each goroutine: its own MapData
			CalculateReading(md, vars, g%2 == 0)
			if math.IsNaN(md.Skills.Reading) {
				t.Errorf("goroutine %d: NaN result", g)
			}
		}()
	}
	wg.Wait()
}

func TestCalculateReading_DenserMapScoresHigher(t *testing.T) {
	vars := DefaultVars()

	sparse := buildFakeMapData(9.0, 60, 300) // objects far apart in time => low density
	CalculateReading(sparse, vars, false)

	dense := buildFakeMapData(9.0, 60, 10) // objects packed close together => high density
	CalculateReading(dense, vars, false)

	if dense.Skills.Reading <= sparse.Skills.Reading {
		t.Fatalf("expected denser map to score higher reading: dense=%v sparse=%v", dense.Skills.Reading, sparse.Skills.Reading)
	}
}

func TestRealMaps(t *testing.T) {
	files, err := getOsuFiles()
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var bm osu.Beatmap
		err = beatmap.Unmarshal(data, &bm)
		if err != nil {
			t.Fatal(err)
		}

		if bm.Mode != osu.ModeOsu {
			t.Skip("skipping non-std map")
		}

		md := NewMapData(&bm, 0)
		PrepareMapData(md)
		CalculateReading(md, DefaultVars(), false)

		if md.Skills.Reading <= 0 || math.IsNaN(md.Skills.Reading) {
			t.Errorf("%s: invalid reading=%f", f, md.Skills.Reading)
		}
	}
}
