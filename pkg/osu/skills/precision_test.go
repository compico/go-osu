package skills

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/beatmap"
)

func TestPrecision_RealMaps(t *testing.T) {
	files, err := getOsuFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Skip("no .osu files found in ./test_data")
	}

	vars := DefaultVars()

	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			var bm osu.Beatmap
			if err := beatmap.Unmarshal(data, &bm); err != nil {
				t.Fatalf("parse error: %v", err)
			}

			if bm.Mode != osu.ModeOsu {
				t.Skip("skipping non-std map")
			}

			md := NewMapData(&bm, 0)
			PrepareMapData(md)
			CalculatePrecision(md, vars)

			if math.IsNaN(md.Skills.Precision) || math.IsInf(md.Skills.Precision, 0) {
				t.Errorf("precision is not finite: %v", md.Skills.Precision)
			}
			if md.Skills.Precision < 0 {
				t.Errorf("precision is negative: %v", md.Skills.Precision)
			}

			if len(md.AimPoints) > 2 && md.Skills.Precision <= 0 {
				t.Errorf("expected positive precision for map with notes, got %v", md.Skills.Precision)
			}
		})
	}
}
