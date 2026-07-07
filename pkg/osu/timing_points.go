package osu

type TimingPoint struct {
	Time                float64
	BeatLength          float64
	Meter               int
	SampleSet           SampleSet
	SampleIndex         int
	Volume              int
	Uninherited         bool
	Effects             int
	PreviousTimingPoint *TimingPoint
}

func (tp *TimingPoint) BPM() float64 {
	if tp.Uninherited {
		if tp.BeatLength <= 0 {
			return 0
		}
		return 60000.0 / tp.BeatLength
	}

	if tp.PreviousTimingPoint != nil {
		return tp.PreviousTimingPoint.BPM()
	}

	return 0
}

func (tp *TimingPoint) SliderVelocity() float64 {
	if tp == nil {
		return 1.0
	}

	if tp.Uninherited {
		return 1.0
	}

	return -100.0 / tp.BeatLength
}
