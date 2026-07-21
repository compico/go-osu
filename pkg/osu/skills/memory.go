package skills

import (
	"math"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

// CalculateMemory портирует memory.cpp's CalculateMemory.
// Оценивает сложность запоминания паттернов карты.
// Требует md.AimPoints, md.Angles, md.Distances (заполняются в prepareMapData).
func CalculateMemory(md *MapData, vars *Vars) float64 {
	totalMemPoints := 0.0
	old := md.Map.HitObjects[0]
	combo := 0

	for i := 1; i < len(md.Map.HitObjects); i++ {
		ho := md.Map.HitObjects[i]

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

		// ВАЖНО: old, а не ho
		if old.Type.IsHitSlider() {
			sliderBonusFactor = vars.Get("Memory", "SliderBuff")
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
						vars.Get("Memory", "FollowpointsNerf") *
						(dist / float64(ho.Time-old.Time))
				}
			}
		}

		if ho.Type.IsHitNormal() || ho.Type.IsHitSpinner() {
			combo++
		} else if ho.Type.IsHitSlider() {
			combo += len(ho.Ticks) + 2
		}

		old = ho
		totalMemPoints += memPoints
	}

	md.Skills.Memory = vars.Get("Memory", "TotalMult") *
		math.Pow(totalMemPoints, vars.Get("Memory", "TotalPow"))

	return md.Skills.Memory
}
func IsObservableFrom(obj osu.HitObject, distance int, fromPos vector2d.Vector2dd) bool {
	dist := obj.Pos.DistanceFrom(fromPos)

	if dist < float64(distance) {
		return true
	}

	return false
}

func GetApproachRelativeSize(time int, hitTime int, ar float64) float64 {
	ms := arToMs(ar)

	if hitTime < time {
		return 0
	}
	if hitTime-ms > time {
		return 0
	}

	diff := hitTime - time
	interval := ms

	return float64(1 + 3*(diff/interval))
}
