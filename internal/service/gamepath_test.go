package service

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestWindowsPathToMounted(t *testing.T) {
	root := filepath.FromSlash("/mnt")

	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{`C:\Users\me\AppData\Local\osu!`, "/mnt/c/Users/me/AppData/Local/osu!", true},
		{`C:/Games/osu!`, "/mnt/c/Games/osu!", true},
		{`d:\osu!\`, "/mnt/d/osu!", true},
		{`C:\`, "/mnt/c", true},
		{`E:`, "/mnt/e", true},
		{"/home/me/osu!", "", false},
		{"relative/osu!", "", false},
		{`\\server\share\osu!`, "", false},
		{"C", "", false},
		{"", "", false},
	}

	for _, tc := range tests {
		got, ok := windowsPathToMounted(tc.in, root)
		if want := filepath.FromSlash(tc.want); ok != tc.ok || got != want {
			t.Errorf("windowsPathToMounted(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, want, tc.ok)
		}
	}
}

func TestNormalizeGamePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("covers the mapping done outside Windows")
	}

	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"   ", ""},
		{`"C:\Games\osu!"`, "/mnt/c/Games/osu!"},
		{`C:\Users\me\AppData\Local\osu!`, "/mnt/c/Users/me/AppData/Local/osu!"},
		{"C:/Games/osu!/", "/mnt/c/Games/osu!"},
		{"/home/me/.wine/drive_c/osu!/", "/home/me/.wine/drive_c/osu!"},
		{"/mnt/c//Games/./osu!", "/mnt/c/Games/osu!"},
	}

	for _, tc := range tests {
		if got := normalizeGamePath(tc.in); got != tc.want {
			t.Errorf("normalizeGamePath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func writeGameDB(t *testing.T, dir string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, gameDBName), nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindGameDir(t *testing.T) {
	appData := []string{"AppData", "Local", "osu!"}

	find := func(mnt, home, winePrefix string) (string, bool) {
		return findGameDir(gamePathCandidates(home, winePrefix, mnt))
	}

	t.Run("windows drive under WSL", func(t *testing.T) {
		mnt, home := t.TempDir(), t.TempDir()
		game := filepath.Join(append([]string{mnt, "c", "Users", "me"}, appData...)...)
		writeGameDB(t, game)

		if got, ok := find(mnt, home, ""); !ok || got != game {
			t.Errorf("got (%q, %v), want %q", got, ok, game)
		}
	})

	t.Run("default wine prefix", func(t *testing.T) {
		mnt, home := t.TempDir(), t.TempDir()
		game := filepath.Join(append([]string{home, ".wine", "drive_c", "users", "me"}, appData...)...)
		writeGameDB(t, game)

		if got, ok := find(mnt, home, ""); !ok || got != game {
			t.Errorf("got (%q, %v), want %q", got, ok, game)
		}
	})

	t.Run("WINEPREFIX", func(t *testing.T) {
		mnt, home, prefix := t.TempDir(), t.TempDir(), t.TempDir()
		game := filepath.Join(append([]string{prefix, "drive_c", "users", "me"}, appData...)...)
		writeGameDB(t, game)

		if got, ok := find(mnt, home, prefix); !ok || got != game {
			t.Errorf("got (%q, %v), want %q", got, ok, game)
		}
	})

	t.Run("osu-wine", func(t *testing.T) {
		mnt, home := t.TempDir(), t.TempDir()
		game := filepath.Join(home, ".local", "share", "osu-wine", "osu!")
		writeGameDB(t, game)

		if got, ok := find(mnt, home, ""); !ok || got != game {
			t.Errorf("got (%q, %v), want %q", got, ok, game)
		}
	})

	t.Run("WSL drive wins over wine", func(t *testing.T) {
		mnt, home := t.TempDir(), t.TempDir()
		wsl := filepath.Join(append([]string{mnt, "d", "Users", "me"}, appData...)...)
		wine := filepath.Join(append([]string{home, ".wine", "drive_c", "users", "me"}, appData...)...)
		writeGameDB(t, wsl)
		writeGameDB(t, wine)

		if got, ok := find(mnt, home, ""); !ok || got != wsl {
			t.Errorf("got (%q, %v), want %q", got, ok, wsl)
		}
	})

	t.Run("skips directories without a database file", func(t *testing.T) {
		mnt, home := t.TempDir(), t.TempDir()

		empty := filepath.Join(append([]string{mnt, "c", "Users", "me"}, appData...)...)
		if err := os.MkdirAll(empty, 0o755); err != nil {
			t.Fatal(err)
		}
		// osu!.db being a directory is not an installation either.
		if err := os.MkdirAll(filepath.Join(home, ".local", "share", "osu-wine", "osu!", gameDBName), 0o755); err != nil {
			t.Fatal(err)
		}

		if got, ok := find(mnt, home, ""); ok {
			t.Errorf("found %q, want nothing", got)
		}
	})

	t.Run("nothing installed", func(t *testing.T) {
		if got, ok := find(t.TempDir(), t.TempDir(), ""); ok {
			t.Errorf("found %q, want nothing", got)
		}
	})
}

// A path from the config used to skip the derived Songs/Skins/Data paths
// entirely, because initGamePath returned as soon as the path was set.
func TestNewOsuServiceDerivesPathsFromConfiguredGamePath(t *testing.T) {
	game := t.TempDir()

	svc, err := NewOsuService(game, discardLogger(), nil)
	if err != nil {
		t.Fatal(err)
	}

	for name, c := range map[string]struct{ got, want string }{
		"game":        {svc.gamePath, game},
		"songs":       {svc.songsPath, filepath.Join(game, "Songs")},
		"skins":       {svc.skinsPath, filepath.Join(game, "Skins")},
		"backgrounds": {svc.backgroundPath, filepath.Join(game, "Data", "bt")},
	} {
		if c.got != c.want {
			t.Errorf("%s path = %q, want %q", name, c.got, c.want)
		}
	}
}

func TestNewOsuServiceMapsWindowsConfigPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("covers the mapping done outside Windows")
	}

	svc, err := NewOsuService(`C:\Games\osu!`, discardLogger(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if want := "/mnt/c/Games/osu!"; svc.gamePath != want {
		t.Errorf("game path = %q, want %q", svc.gamePath, want)
	}
	if want := "/mnt/c/Games/osu!/Songs"; svc.songsPath != want {
		t.Errorf("songs path = %q, want %q", svc.songsPath, want)
	}
}
