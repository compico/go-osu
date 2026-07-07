package skills

import (
	"fmt"

	"github.com/compico/go-osu/pkg/osu"
)

type SkillResult struct {
	Mods       osu.Mod
	ModsString string
	Skills     Skills
}

func CalculateAllSkillsForMods(bm *osu.Beatmap, mods osu.Mod, vars *Vars) (*SkillResult, error) {
	// Применяем моды к копии карты
	modifiedBm := ApplyMods(bm, mods)

	// Создаём независимый контекст расчёта
	md := NewMapData(modifiedBm, mods)

	// Подготовка данных (геометрия слайдеров, углы, интервалы)
	PrepareMapData(md)

	// Рассчитываем все 8 скиллов
	CalculateStamina(md, vars)
	CalculateTenacity(md, vars)
	CalculateAimStrains(md, vars)
	CalculateAgility(md, vars)
	CalculatePrecision(md, vars)
	CalculateReading(md, vars, mods&osu.HD != 0) // Hidden влияет на Reading
	CalculateReaction(md, vars)
	CalculateMemory(md, vars)
	CalculateAccuracy(md, vars)

	return &SkillResult{
		Mods:       mods,
		ModsString: modsToString(mods),
		Skills:     md.Skills,
	}, nil
}

func CalculateAllSkillsFromFile(bm osu.Beatmap, modsCombination []osu.Mod) (map[osu.Mod]*SkillResult, error) {
	vars := DefaultVars()
	results := make(map[osu.Mod]*SkillResult)

	for _, mods := range modsCombination {
		result, err := CalculateAllSkillsForMods(&bm, mods, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate for mods %v: %w", mods, err)
		}
		results[mods] = result
	}

	return results, nil
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

	combinations := make([]osu.Mod, 0, 57)

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
