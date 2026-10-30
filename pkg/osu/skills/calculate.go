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

func (md *MapData) calculateSkills(vars *Vars) {
	md.calculateReaction(vars, md.HasMod(osu.HD))
	md.calculateStamina(vars)
	md.calculateTenacity(vars)
	md.calculateAgility(vars)
	md.calculatePrecision(vars)
	md.calculateAccuracy(vars)
	if md.HasMod(osu.FL) {
		md.calculateMemory(vars)
	}
	md.calculateReading(vars, md.HasMod(osu.HD))
}
