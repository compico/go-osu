package skills

import (
	"math"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

// CalculateMemory портирует memory.cpp's CalculateMemory.
// Оценивает сложность запоминания паттернов карты.
// Требует данные карты, заполненные на этапе подготовки.
func CalculateMemory(md *MapData, vars *Vars) float64 {
	return md.calculateMemory(vars)
}

func (md *MapData) calculateMemory(vars *Vars) float64 {
	totalMemPoints := 0.0
	md.MemoryTotals = make([]float64, 0, len(md.Map.HitObjects))
	md.MemoryTimes = make([]int, 0, len(md.Map.HitObjects))
	old := md.Map.HitObjects[0]
	combo := 0

	debugf("[MEMORY] hitObjectCount=%v\n", len(md.Map.HitObjects))
	for i := 1; i < len(md.Map.HitObjects); i++ {
		ho := md.Map.HitObjects[i]
		if ho.Type.IsHitSpinner() {
			continue
		}

		memPoints := 0.0
		observableDist := 160

		if combo < 100 {
			observableDist = 160
		} else if combo < 200 {
			observableDist = 120
		} else {
			observableDist = 100
		}

		sliderBonusFactor := 1.0

		if old.Type.IsHitSlider() {
			sliderBonusFactor = vars.Memory.SliderBuff
		}

		observable := false
		helpPixels := 0

		for j := i - 1; j > 0; j-- {
			prev := md.Map.HitObjects[j]

			if ho.Time-prev.Time > arToMs(md.Map.ApproachRate) {
				break
			}

			if !md.HasMod(osu.HD) {
				size := GetApproachRelativeSize(
					prev.EndTime,
					ho.Time,
					md.Map.ApproachRate,
				)

				helpPixels = int(size * float64(cs2px(md.Map.CircleSize)))
			} else {
				observableTime := ho.Time -
					int(float64(arToMs(md.Map.ApproachRate))*0.3)

				if prev.Time > observableTime {
					continue
				}

				helpPixels = cs2px(md.Map.CircleSize)
			}

			if IsObservableFrom(
				ho,
				observableDist+helpPixels,
				prev.Pos,
			) {
				observable = true
				break
			}
		}

		if !observable {
			if !md.HasMod(osu.HD) {
				size := GetApproachRelativeSize(
					old.EndTime,
					ho.Time,
					md.Map.ApproachRate,
				)

				helpPixels = int(size * float64(cs2px(md.Map.CircleSize)))
			} else {
				helpPixels = cs2px(md.Map.CircleSize)
			}

			dist := ho.Pos.DistanceFrom(old.EndPoint)

			if ho.Type.IsHitNewCombo() || ho.Type.IsHitColourHax() {
				if dist > float64(observableDist+helpPixels) {
					memPoints = sliderBonusFactor *
						(dist / float64(ho.Time-old.Time))
				}
			} else {
				if dist > float64(observableDist+helpPixels) {
					memPoints = sliderBonusFactor *
						vars.Memory.FollowpointsNerf *
						(dist / float64(ho.Time-old.Time))
				}
			}
			debugf("[MEMORY-DIST] i=%v curPos=(%v,%v) oldEndPoint=(%v,%v) oldType=%v dist=%v\n",
				i, ho.Pos.X, ho.Pos.Y, old.EndPoint.X, old.EndPoint.Y, old.Type, ho.Pos.DistanceFrom(old.EndPoint))
		}

		if ho.Type.IsHitNormal() || ho.Type.IsHitSpinner() {
			combo++
		} else if ho.Type.IsHitSlider() {
			combo += len(ho.Ticks) + 2
		}

		old = ho
		totalMemPoints += memPoints
		md.MemoryTotals = append(md.MemoryTotals, totalMemPoints)
		md.MemoryTimes = append(md.MemoryTimes, ho.Time)

		debugf("[MEMORY] i=%v combo=%v observableDist=%v sliderBonus=%v observable=%v helpPixels=%v memPoints=%v totalMemPoints=%v hitObjectType=%v\n",
			i, combo, observableDist, sliderBonusFactor, map[bool]int{false: 0, true: 1}[observable], helpPixels, memPoints, totalMemPoints, ho.Type)
	}

	md.Skills.Memory = memoryScale(totalMemPoints, vars)

	return md.Skills.Memory
}

// memoryScale converts a total of memory points to the memory skill scale.
// The skill is the scale of the total over the whole map, so the same
// conversion applied to the running total gives a curve that ends at the skill.
func memoryScale(total float64, vars *Vars) float64 {
	return vars.Memory.TotalMult * math.Pow(total, vars.Memory.TotalPow)
}

func IsObservableFrom(obj osu.HitObject, distance int, fromPos vector2d.Vector2dd) bool {
	dist := obj.Pos.DistanceFrom(fromPos)

	if dist < float64(distance) {
		return true
	}

	return false
}

func GetApproachRelativeSize(time int, hitTime int, ar float64) float64 {
	if hitTime < time {
		return 1
	}
	if hitTime-arToMs(ar) > time {
		return 0
	}

	diff := float64(hitTime - time)
	interval := float64(arToMs(ar))

	return 1 + 3*(diff/interval)
}
