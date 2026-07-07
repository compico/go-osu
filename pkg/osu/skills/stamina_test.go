package skills

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/beatmap"
)

func TestStamina_RealMaps(t *testing.T) {
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
			CalculateStamina(md, vars)

			// Базовые проверки на валидность результата
			if math.IsNaN(md.Skills.Stamina) || math.IsInf(md.Skills.Stamina, 0) {
				t.Errorf("stamina is not finite: %v", md.Skills.Stamina)
			}
			if md.Skills.Stamina < 0 {
				t.Errorf("stamina is negative: %v", md.Skills.Stamina)
			}

			// Для карт с >1 объектом ожидаем положительный скилл
			if len(md.PressIntervals) > 1 && md.Skills.Stamina <= 0 {
				t.Errorf("expected positive stamina for map with notes, got %v", md.Skills.Stamina)
			}
		})
	}
}
