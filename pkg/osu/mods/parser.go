package mods

import (
	"fmt"
	"strings"

	"github.com/compico/go-osu/pkg/osu"
)

var modAbbreviations = []struct {
	Name string
	Mod  osu.Mod
}{
	{"NF", osu.NF}, {"EZ", osu.EZ}, {"HD", osu.HD}, {"HR", osu.HR},
	{"SD", osu.SD}, {"DT", osu.DT}, {"RL", osu.RL}, {"HT", osu.HT},
	{"FL", osu.FL}, {"AU", osu.AU}, {"SO", osu.SO}, {"AP", osu.AP},
}

// Parse parses a concatenated mod abbreviation string such as
// "HDDT" or "hddt" into a Mod bitmask — the reverse of modsToString. An
// empty string, "NM", "NOMOD", or "NONE" (case-insensitive) returns 0.
// Returns an error on any chunk that isn't a recognized 2-letter mod code.
func Parse(s string) (osu.Mod, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" || s == "NM" || s == "NOMOD" || s == "NONE" {
		return 0, nil
	}
	if len(s)%2 != 0 {
		return 0, fmt.Errorf("mods string %q has an odd length, expected pairs of 2-letter mod codes", s)
	}

	var mods osu.Mod
	for i := 0; i < len(s); i += 2 {
		chunk := s[i : i+2]
		found := false
		for _, m := range modAbbreviations {
			if m.Name == chunk {
				mods |= m.Mod
				found = true
				break
			}
		}
		if !found {
			return 0, fmt.Errorf("unknown mod abbreviation %q in %q", chunk, s)
		}
	}

	return mods, nil
}
