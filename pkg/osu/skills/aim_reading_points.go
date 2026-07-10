package skills

import (
	"math"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

// populateReadingPoints constructs the ReadingPoint slice from AimPoints,
// Angles, Distances, and map properties (AR).
func populateReadingPoints(md *MapData) {
	ar := md.Map.ApproachRate
	preemptMs := arToMs(ar)

	var readingPoints []ReadingPoint
	var previousPos *vector2d.Vector2dd
	for i, ap := range md.AimPoints {
		preemptTime := ap.Time - int(preemptMs)
		fadeInTime := preemptTime
		fadeOutTime := ap.Time

		if i < len(md.Map.HitObjects) {
			ho := md.Map.HitObjects[i]
			vis := getVisibilityTimes(&ho, ar, md.HasMod(osu.HD), 0.0, 1.0)
			fadeInTime = vis.first
			fadeOutTime = vis.second
		}

		// The original C++ pipeline attaches the angle of the triplet centered
		// at aim point i to reading point i+2, so the first two reading points
		// keep a zero angle and later points use the matching angle slot.
		angle := 0.0
		if i >= 2 && i-2 < len(md.Angles) {
			angle = md.Angles[i-2]
		}

		dist := 0.0
		if i < len(md.Distances) {
			dist = md.Distances[i]
		}
		if previousPos != nil {
			dist = ap.Pos.DistanceFrom(*previousPos)
		} else {
			dist = 0
		}

		readingPoints = append(readingPoints, ReadingPoint{
			Index:   i,
			Time:    ap.Time,
			Preempt: preemptTime,
			FadeIn:  fadeInTime,
			FadeOut: fadeOutTime,
			Angle:   angle,
			Pos:     ap.Pos,
			Dist:    dist,
		})
		previousPos = &ap.Pos
	}

	md.ReadingPoints = readingPoints
}

func getVisibilityTimes(ho *osu.HitObject, ar float64, hidden bool, opacityStart, opacityEnd float64) struct{ first, second int } {
	preemptTime := ho.Time - int(arToMs(ar))
	if hidden {
		fadeInDuration := 0.4 * arToMs(ar)
		fadeInTimeEnd := float64(preemptTime) + fadeInDuration
		first := int(getValue(float64(preemptTime), fadeInTimeEnd, opacityStart))
		second := ho.Time
		if ho.Type&int(osu.HitSlider) != 0 {
			fadeOutDuration := 0.7 * (float64(ho.Time) - fadeInTimeEnd)
			fadeOutTimeEnd := fadeInTimeEnd + fadeOutDuration
			second = int(getValue(fadeInTimeEnd, fadeOutTimeEnd, 1.0-opacityStart))
		} else {
			endTime := ho.Time
			if ho.EndTime > 0 {
				endTime = ho.EndTime
			}
			fadeOutDuration := float64(endTime) - fadeInTimeEnd
			fadeOutTimeEnd := fadeInTimeEnd + fadeOutDuration
			second = int(getValue(fadeInTimeEnd, fadeOutTimeEnd, 1.0-opacityEnd))
		}
		return struct{ first, second int }{first: first, second: second}
	}

	fadeInDuration := math.Min(arToMs(ar), 400.0)
	fadeInTimeEnd := float64(preemptTime) + fadeInDuration
	first := int(getValue(float64(preemptTime), fadeInTimeEnd, opacityStart))
	second := ho.Time
	if ho.Type&int(osu.HitSlider) != 0 {
		second = ho.EndTime
	}
	return struct{ first, second int }{first: first, second: second}
}
