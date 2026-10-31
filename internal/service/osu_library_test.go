package service

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/compico/go-osu/pkg/osu"
)

const (
	envGamePath = "OSU_PATH"
	envSample   = "OSU_SAMPLE"

	defaultSample    = 300
	maxReported      = 15
	maxProblemsRatio = 0.05
)

// TestLibraryPaths runs the path handling of the Osu service against a real
// osu! installation: it resolves the game directory the way the app does,
// reads osu!.db, and checks that for a sample of beatmaps the .osu file
// exists and parses and that the audio file exists. It prints everything
// needed to see where paths go wrong (OS, resolved directories, the first
// problems) and is skipped unless OSU_PATH is set.
//
// Run it from the repository root, for example under WSL:
//
//	OSU_PATH=/mnt/c/Users/me/AppData/Local/osu! go test ./internal/service -run TestLibraryPaths -v -count=1
//
// A Windows-style value (OSU_PATH='C:\Users\me\AppData\Local\osu!') works as
// well, and checks the same mapping osu.path in the config goes through.
// OSU_SAMPLE sets how many beatmaps to check (default 300, "all" for every
// one).
func TestLibraryPaths(t *testing.T) {
	raw := os.Getenv(envGamePath)
	if raw == "" {
		t.Skipf("%s is not set; point it at the osu! game directory to run this check", envGamePath)
	}

	t.Logf("platform: %s/%s, %s", runtime.GOOS, runtime.GOARCH, runtime.Version())
	if data, err := os.ReadFile("/proc/version"); err == nil {
		t.Logf("/proc/version: %s", strings.TrimSpace(string(data)))
	}
	t.Logf("%s as given: %q", envGamePath, raw)

	svc, err := NewOsuService(raw, discardLogger(), nil)
	if err != nil {
		t.Fatalf("NewOsuService: %v", err)
	}

	t.Logf("game path:   %q: %s", svc.gamePath, describePath(svc.gamePath))
	t.Logf("osu!.db:     %s", describePath(filepath.Join(svc.gamePath, gameDBName)))
	t.Logf("Songs:       %q: %s", svc.songsPath, describePath(svc.songsPath))
	t.Logf("Skins:       %q: %s", svc.skinsPath, describePath(svc.skinsPath))
	t.Logf("backgrounds: %q: %s", svc.backgroundPath, describePath(svc.backgroundPath))

	if err := svc.ReadOsuDBFile(); err != nil {
		t.Fatalf("ReadOsuDBFile: %v", err)
	}
	all := svc.GetBeatmaps()

	// The same subset Syncer.mapRawBeatmaps works on.
	var maps []osu.DatabaseBeatmap
	for _, bm := range all {
		if osu.GameMode(bm.Mode) != osu.ModeOsu || bm.BeatmapID <= 0 || bm.BeatmapSetID <= 0 {
			continue
		}
		maps = append(maps, bm)
	}
	t.Logf("osu!.db: %d beatmaps, %d of them osu!standard maps the sync works on", len(all), len(maps))
	if len(maps) == 0 {
		t.Fatal("nothing to check: osu!.db has no osu!standard beatmaps")
	}

	sample := sampleSize(len(maps))
	step := max(1, len(maps)/sample)

	var (
		checked, osuMissing, parseFailed, audioMissing, backslashed int
		problems                                                    []string
	)
	report := func(format string, args ...any) {
		if len(problems) < maxReported {
			problems = append(problems, fmt.Sprintf(format, args...))
		}
	}

	for i := 0; i < len(maps) && checked < sample; i += step {
		bm := maps[i]
		checked++

		if strings.ContainsRune(bm.FolderName+bm.AudioFileName+bm.NameOfTheOsuFile, '\\') {
			backslashed++
			report("beatmap %d: name from osu!.db contains a backslash: folder=%q audio=%q osu file=%q",
				bm.BeatmapID, bm.FolderName, bm.AudioFileName, bm.NameOfTheOsuFile)
		}

		osuPath := svc.GetOsuFilePath(bm)
		if _, err := os.Stat(osuPath); err != nil {
			osuMissing++
			report("beatmap %d: .osu file is missing: %q (%v)", bm.BeatmapID, osuPath, err)
		} else if _, err := svc.ParseBeatmapFile(bm); err != nil {
			parseFailed++
			report("beatmap %d: %v", bm.BeatmapID, err)
		}

		if bm.AudioFileName != "" {
			audioPath := filepath.Join(svc.songsPath, bm.FolderName, bm.AudioFileName)
			if _, err := os.Stat(audioPath); err != nil {
				audioMissing++
				report("beatmap %d: audio file is missing: %q (%v)", bm.BeatmapID, audioPath, err)
			}
		}
	}

	t.Logf("checked %d beatmaps (every %d-th): .osu missing %d, .osu failed to parse %d, audio missing %d, names with a backslash %d",
		checked, step, osuMissing, parseFailed, audioMissing, backslashed)
	for _, p := range problems {
		t.Logf("  %s", p)
	}

	// A few stale entries are normal (osu!.db can list maps that were deleted
	// from Songs); broken paths affect nearly everything.
	for name, n := range map[string]int{".osu file missing": osuMissing, ".osu parse failure": parseFailed, "audio file missing": audioMissing} {
		if float64(n) > maxProblemsRatio*float64(checked) {
			t.Errorf("%s for %d of %d checked beatmaps", name, n, checked)
		}
	}
}

// sampleSize returns how many beatmaps to check out of total.
func sampleSize(total int) int {
	v := strings.TrimSpace(os.Getenv(envSample))
	if strings.EqualFold(v, "all") {
		return total
	}

	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		n = defaultSample
	}

	return min(n, total)
}

// describePath says what is at path, for the diagnostic log.
func describePath(path string) string {
	info, err := os.Stat(path)
	switch {
	case err != nil:
		return "not accessible: " + err.Error()
	case info.IsDir():
		return "directory"
	default:
		return fmt.Sprintf("file, %d bytes", info.Size())
	}
}
