package skills

import (
	"github.com/compico/go-osu/pkg/osu"
)

type SkillResult struct {
	Mods       osu.Mod
	ModsString string
	Skills     Skills
}

// DefaultModCombinations возвращает стандартный набор комбинаций модов,
// для которых обычно считают скиллы.
func DefaultModCombinations() []osu.Mod {
	base := []osu.Mod{
		osu.EZ,
		osu.HT,
		osu.DT,
		osu.HD,
		osu.HR,
		osu.FL,
	}

	combinations := make([]osu.Mod, 0, 36)

	// Перебираем все подмножества
	for mask := 0; mask < (1 << len(base)); mask++ {
		var mods osu.Mod

		for i, mod := range base {
			if mask&(1<<i) != 0 {
				mods |= mod
			}
		}

		// EZ несовместим с HR
		if mods&osu.EZ != 0 && mods&osu.HR != 0 {
			continue
		}

		// DT несовместим с HT
		if mods&osu.DT != 0 && mods&osu.HT != 0 {
			continue
		}

		combinations = append(combinations, mods)
	}

	return combinations
}

// modsToString преобразует флаг модов в человекочитаемую строку.
func modsToString(mods osu.Mod) string {
	if mods == 0 {
		return "None"
	}
	s := ""
	if mods&osu.NF != 0 {
		s += "NF"
	}
	if mods&osu.EZ != 0 {
		s += "EZ"
	}
	if mods&osu.HD != 0 {
		s += "HD"
	}
	if mods&osu.HR != 0 {
		s += "HR"
	}
	if mods&osu.DT != 0 {
		s += "DT"
	}
	if mods&osu.HT != 0 {
		s += "HT"
	}
	if mods&osu.FL != 0 {
		s += "FL"
	}
	return s
}

func calculateSkills(md *MapData, vars *Vars) {
	CalculateReaction(md, vars, md.HasMod(osu.HD))
	CalculateStamina(md, vars)
	CalculateTenacity(md, vars)
	CalculateAgility(md, vars)
	CalculatePrecision(md, vars)
	CalculateAccuracy(md, vars)
	if md.HasMod(osu.FL) {
		CalculateMemory(md, vars)
	}
	CalculateReading(md, vars, md.HasMod(osu.HD))
}
