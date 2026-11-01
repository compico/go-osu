package skills

import (
	"math"
	"sort"
)

// CalculateTenacity портирует tenacity.cpp's CalculateTenacity.
// Рассчитывает способность к быстрому тапу (streams и bursts).
// Требует md.Streams, заполненный на этапе подготовки карты.
func CalculateTenacity(md *MapData, vars *Vars) {
	md.calculateTenacity(vars)
}

func (md *MapData) calculateTenacity(vars *Vars) {
	longestStream := getLongestStream(md.Streams)

	debugf("[TENACITY] longestStream: interval=%v length=%v\n", longestStream.Interval, longestStream.Length)

	if longestStream.Length == 0 {
		debugf("[TENACITY] longestStream.Length == 0, tenacity=0\n")
		md.Skills.Tenacity = 0
		return
	}

	md.Skills.Tenacity = tenacityOfStream(longestStream.Interval, longestStream.Length, vars)
}

// tenacityOfStream is the tenacity skill of a map whose longest stream has the
// given press interval (ms) and length (notes).
func tenacityOfStream(interval, length int, vars *Vars) float64 {
	// intervalScaled = 1.0 / pow(interval, pow(interval, IntervalPow) * IntervalMult) * IntervalMult2
	intervalPow := vars.Tenacity.IntervalPow
	intervalMult := vars.Tenacity.IntervalMult
	intervalMult2 := vars.Tenacity.IntervalMult2

	exponent := math.Pow(float64(interval), intervalPow) * intervalMult
	intervalScaled := 1.0 / math.Pow(float64(interval), exponent) * intervalMult2

	debugf("[TENACITY] IntervalPow=%v IntervalMult=%v IntervalMult2=%v exponent=%v intervalScaled=%v\n",
		intervalPow, intervalMult, intervalMult2, exponent, intervalScaled)

	// lengthScaled = pow(LengthDivisor / length, (LengthDivisor / length) * LengthMult)
	lengthDivisor := vars.Tenacity.LengthDivisor
	lengthMult := vars.Tenacity.LengthMult

	lengthBase := lengthDivisor / float64(length)
	lengthScaled := math.Pow(lengthBase, lengthBase*lengthMult)

	debugf("[TENACITY] LengthDivisor=%v LengthMult=%v lengthBase=%v lengthScaled=%v\n",
		lengthDivisor, lengthMult, lengthBase, lengthScaled)

	// tenacity = intervalScaled * lengthScaled
	tenacity := intervalScaled * lengthScaled

	debugf("[TENACITY] tenacity(before totalMult/Pow)=%v\n", tenacity)

	// Применяем финальный множитель
	totalMult := vars.Tenacity.TotalMult
	totalPow := vars.Tenacity.TotalPow
	result := totalMult * math.Pow(tenacity, totalPow)

	debugf("[TENACITY] TotalMult=%v TotalPow=%v final=%v\n", totalMult, totalPow, result)
	return result
}

// getLongestStream returns the first interval bucket, in ascending interval
// order, that contains a stream longer than one note. It ports
// GetLongestStream from tenacity.cpp.
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
