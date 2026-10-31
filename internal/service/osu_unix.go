//go:build darwin || freebsd || linux || netbsd || openbsd

package service

import (
	"errors"
	"os"
)

// detectGamePath looks for osu!stable in the places it usually lives when
// the app does not run on Windows (WSL, Wine and the tools built on it).
// osu.path in the config always takes precedence over this.
func detectGamePath() (string, error) {
	home, _ := os.UserHomeDir()

	candidates := gamePathCandidates(home, os.Getenv("WINEPREFIX"), wslMountRoot)
	if dir, ok := findGameDir(candidates); ok {
		return dir, nil
	}

	return "", errors.New("osu! installation not found in the usual places (WSL drives, Wine, osu-wine, Lutris, Bottles): set osu.path in the config")
}
