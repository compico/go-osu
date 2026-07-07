package skills

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/beatmap"
)

func TestAgility_RealMaps(t *testing.T) {
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
			CalculateAimStrains(md, vars)
			CalculateAgility(md, vars)

			if math.IsNaN(md.Skills.Agility) || math.IsInf(md.Skills.Agility, 0) {
				t.Errorf("agility is not finite: %v", md.Skills.Agility)
			}
			if md.Skills.Agility < 0 {
				t.Errorf("agility is negative: %v", md.Skills.Agility)
			}

			// Для карт с >2 объектов ожидаем положительный скилл
			if len(md.AimStrains) > 2 && md.Skills.Agility <= 0 {
				t.Errorf("expected positive agility for map with notes, got %v", md.Skills.Agility)
			}
		})
	}
}
