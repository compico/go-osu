package skills

import (
	"math"
)

// CalculateTenacity портирует tenacity.cpp's CalculateTenacity.
// Рассчитывает способность к быстрому тапу (streams и bursts).
// Требует md.Streams (заполняется в prepareMapData).
func CalculateTenacity(md *MapData, vars *Vars) {
	longestStream := getLongestStream(md.Streams)

	if longestStream.Length == 0 {
		md.Skills.Tenacity = 0
		return
	}

	// intervalScaled = 1.0 / pow(interval, pow(interval, IntervalPow) * IntervalMult) * IntervalMult2
	intervalPow := vars.Get("Tenacity", "IntervalPow")
	intervalMult := vars.Get("Tenacity", "IntervalMult")
	intervalMult2 := vars.Get("Tenacity", "IntervalMult2")

	exponent := math.Pow(float64(longestStream.Interval), intervalPow) * intervalMult
	intervalScaled := 1.0 / math.Pow(float64(longestStream.Interval), exponent) * intervalMult2

	// lengthScaled = pow(LengthDivisor / length, (LengthDivisor / length) * LengthMult)
	lengthDivisor := vars.Get("Tenacity", "LengthDivisor")
	lengthMult := vars.Get("Tenacity", "LengthMult")

	lengthBase := lengthDivisor / float64(longestStream.Length)
	lengthScaled := math.Pow(lengthBase, lengthBase*lengthMult)

	// tenacity = intervalScaled * lengthScaled
	tenacity := intervalScaled * lengthScaled

	// Применяем финальный множитель
	totalMult := vars.Get("Tenacity", "TotalMult")
	totalPow := vars.Get("Tenacity", "TotalPow")
	md.Skills.Tenacity = totalMult * math.Pow(tenacity, totalPow)
}

// getLongestStream находит самый длинный stream в map.
// Портирует GetLongestStream из tenacity.cpp.
func getLongestStream(streams map[int][][]int) Stream {
	mx := 1
	interval := 0

	for key, stream := range streams {
		interval = key

		mx = 1
		for _, j := range stream {
			length := len(j) + 1
			if length > mx {
				mx = length
			}
		}

		if mx > 1 {
			break
		}
	}

	return Stream{interval, mx}
}
