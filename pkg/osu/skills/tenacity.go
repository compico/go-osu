package skills

import (
	"math"
	"sort"
)

// CalculateTenacity портирует tenacity.cpp's CalculateTenacity.
// Рассчитывает способность к быстрому тапу (streams и bursts).
// Требует md.Streams (заполняется в prepareMapData).
func CalculateTenacity(md *MapData, vars *Vars) {
	longestStream := getLongestStream(md.Streams)

	debugf("[TENACITY] longestStream: interval=%v length=%v\n", longestStream.Interval, longestStream.Length)

	if longestStream.Length == 0 {
		debugf("[TENACITY] longestStream.Length == 0, tenacity=0\n")
		md.Skills.Tenacity = 0
		return
	}

	// intervalScaled = 1.0 / pow(interval, pow(interval, IntervalPow) * IntervalMult) * IntervalMult2
	intervalPow := vars.Get("Tenacity", "IntervalPow")
	intervalMult := vars.Get("Tenacity", "IntervalMult")
	intervalMult2 := vars.Get("Tenacity", "IntervalMult2")

	exponent := math.Pow(float64(longestStream.Interval), intervalPow) * intervalMult
	intervalScaled := 1.0 / math.Pow(float64(longestStream.Interval), exponent) * intervalMult2

	debugf("[TENACITY] IntervalPow=%v IntervalMult=%v IntervalMult2=%v exponent=%v intervalScaled=%v\n",
		intervalPow, intervalMult, intervalMult2, exponent, intervalScaled)

	// lengthScaled = pow(LengthDivisor / length, (LengthDivisor / length) * LengthMult)
	lengthDivisor := vars.Get("Tenacity", "LengthDivisor")
	lengthMult := vars.Get("Tenacity", "LengthMult")

	lengthBase := lengthDivisor / float64(longestStream.Length)
	lengthScaled := math.Pow(lengthBase, lengthBase*lengthMult)

	debugf("[TENACITY] LengthDivisor=%v LengthMult=%v lengthBase=%v lengthScaled=%v\n",
		lengthDivisor, lengthMult, lengthBase, lengthScaled)

	// tenacity = intervalScaled * lengthScaled
	tenacity := intervalScaled * lengthScaled

	debugf("[TENACITY] tenacity(before totalMult/Pow)=%v\n", tenacity)

	// Применяем финальный множитель
	totalMult := vars.Get("Tenacity", "TotalMult")
	totalPow := vars.Get("Tenacity", "TotalPow")
	md.Skills.Tenacity = totalMult * math.Pow(tenacity, totalPow)

	debugf("[TENACITY] TotalMult=%v TotalPow=%v final=%v\n", totalMult, totalPow, md.Skills.Tenacity)
}

// getLongestStream находит самый длинный stream в map.
// Портирует GetLongestStream из tenacity.cpp.
func getLongestStream(streams map[int][][]int) Stream {
	mx := 1
	interval := 0

	keys := make([]int, 0, len(streams))
	for k := range streams {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, key := range keys {
		interval = key
		stream := streams[key]

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
