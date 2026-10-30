package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/osu/beatmap"
)

// BenchmarkProcessBeatmap measures the full calculation on the largest
// standard-mode fixture in test_data. Parsing is outside the timed section so
// the result describes skills calculation rather than file decoding.
func BenchmarkProcessBeatmap(b *testing.B) {
	files, err := getOsuFiles()
	if err != nil {
		b.Fatal(err)
	}

	var largest osu.Beatmap
	largestFile := ""
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			b.Fatal(err)
		}

		var candidate osu.Beatmap
		if err := beatmap.Unmarshal(data, &candidate); err != nil {
			b.Fatalf("parse %s: %v", file, err)
		}
		if candidate.Mode != osu.ModeOsu || len(candidate.HitObjects) <= len(largest.HitObjects) {
			continue
		}

		largest = candidate
		largestFile = file
	}
	if largestFile == "" {
		b.Skip("no standard-mode beatmap fixtures in ./test_data")
	}

	vars := NewVars()
	mods := []struct {
		name string
		mods osu.Mod
	}{
		{name: "NM", mods: osu.NM},
		{name: "HDDT", mods: osu.HD | osu.DT},
	}

	for _, tc := range mods {
		b.Run(fmt.Sprintf("%s/%s", filepath.Base(largestFile), tc.name), func(b *testing.B) {
			b.ReportAllocs()
			b.ReportMetric(float64(len(largest.HitObjects)), "hitobjects/op")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ProcessBeatmap(&largest, tc.mods, vars)
			}
		})
	}
}
