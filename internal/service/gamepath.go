package service

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// gameDBName is the osu!stable database. A directory that contains it is
// taken to be an osu! installation.
const gameDBName = "osu!.db"

// wslMountRoot is where WSL mounts Windows drives by default (/mnt/c, /mnt/d, ...).
const wslMountRoot = "/mnt"

// normalizeGamePath turns a game directory as the user wrote it into a path
// that works on the OS the app runs on. A config copied from Windows holds
// something like C:\Users\me\AppData\Local\osu! or C:/Games/osu!; outside
// Windows such a path is mapped to where WSL mounts that drive
// (/mnt/c/Users/me/...). On Windows the path is only cleaned.
func normalizeGamePath(p string) string {
	p = strings.Trim(strings.TrimSpace(p), `"'`)
	if p == "" {
		return ""
	}

	if runtime.GOOS != "windows" {
		if mapped, ok := windowsPathToMounted(p, wslMountRoot); ok {
			p = mapped
		}
	}

	return filepath.Clean(p)
}

// windowsPathToMounted maps a Windows drive path (C:\dir or C:/dir) to the
// same location under mountRoot (mountRoot/c/dir). It reports false for
// anything that does not start with a drive letter, such as a path that is
// already native to the current OS.
func windowsPathToMounted(p, mountRoot string) (string, bool) {
	if len(p) < 2 || p[1] != ':' || !isASCIILetter(p[0]) {
		return "", false
	}

	drive := strings.ToLower(p[:1])
	rest := strings.TrimLeft(strings.ReplaceAll(p[2:], `\`, "/"), "/")

	return filepath.Join(mountRoot, drive, rest), true
}

func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// gamePathCandidates lists glob patterns for the places osu!stable usually
// lives when the app does not run on Windows, most likely first: the Windows
// side of a WSL install, then the usual Wine setups (osu-wine, a plain Wine
// prefix, Lutris, Bottles). Patterns that match nothing are harmless.
func gamePathCandidates(home, winePrefix, mountRoot string) []string {
	var patterns []string
	add := func(elem ...string) {
		patterns = append(patterns, filepath.Join(elem...))
	}

	// WSL: osu! on a Windows drive. The installer's default is %LOCALAPPDATA%\osu!.
	add(mountRoot, "*", "Users", "*", "AppData", "Local", "osu!")
	add(mountRoot, "*", "Games", "osu!")
	add(mountRoot, "*", "osu!")

	var prefixes []string
	if winePrefix != "" {
		prefixes = append(prefixes, winePrefix)
	}
	if home != "" {
		// osu-wine keeps the game outside of the Wine prefix.
		add(home, ".local", "share", "osu-wine", "osu!")

		prefixes = append(prefixes,
			filepath.Join(home, ".wine"),
			filepath.Join(home, "Games", "*"), // Lutris
			filepath.Join(home, ".local", "share", "bottles", "bottles", "*"),
			filepath.Join(home, ".var", "app", "com.usebottles.bottles", "data", "bottles", "bottles", "*"),
		)
	}
	for _, prefix := range prefixes {
		add(prefix, "drive_c", "users", "*", "AppData", "Local", "osu!")
		add(prefix, "drive_c", "users", "*", "Local Settings", "Application Data", "osu!") // older Wine
	}

	return patterns
}

// findGameDir returns the first directory matched by one of the patterns
// that holds osu!.db.
func findGameDir(patterns []string) (string, bool) {
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, dir := range matches {
			if isGameDir(dir) {
				return dir, true
			}
		}
	}

	return "", false
}

func isGameDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, gameDBName))
	return err == nil && info.Mode().IsRegular()
}
