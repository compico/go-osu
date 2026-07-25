package osu

import "math"

type TimingPoint struct {
	Time        float64
	BeatLength  float64
	Meter       int
	SampleSet   SampleSet
	SampleIndex int
	Volume      int
	Uninherited bool
	Effects     int
}

func (b *Beatmap) BPM() struct {
	Min float64
	Max float64
	Avg float64
} {
	result := struct {
		Min float64
		Max float64
		Avg float64
	}{
		Min: math.MaxFloat64,
	}

	var sum float64
	var count int

	for _, tp := range b.TimingPoints {
		// Только красные линии задают BPM
		if !tp.Uninherited || tp.BeatLength <= 0 {
			continue
		}

		bpm := 60000.0 / tp.BeatLength

		if bpm < result.Min {
			result.Min = bpm
		}

		if bpm > result.Max {
			result.Max = bpm
		}

		sum += bpm
		count++
	}

	if count == 0 {
		return struct {
			Min float64
			Max float64
			Avg float64
		}{}
	}

	result.Avg = sum / float64(count)

	return result
}
