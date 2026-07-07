package skills

// gatherAimAndReadingPoints processes the map's HitObjects to populate
// md.AimPoints and md.ReadingPoints. This is the crucial preparation step
// required before calling CalculateReading or any other skill calculation
// that depends on these points.
func gatherAimAndReadingPoints(md *MapData) {
	gatherAimPoints(md)
	calculateAngles(md)
	populateReadingPoints(md)
}

// calculateAngles computes the directional angle for each AimPoint
// (except the first and last) and stores them in md.Angles.
// It also populates md.Distances.
func calculateAngles(md *MapData) {
	n := len(md.AimPoints)
	if n == 0 {
		return
	}

	angles := make([]float64, n)
	distances := make([]float64, n)

	// First point has no previous, distance is 0
	angles[0] = 0.0
	distances[0] = 0.0

	for i := 1; i < n; i++ {
		prevPos := md.AimPoints[i-1].Pos
		currPos := md.AimPoints[i].Pos
		distances[i] = currPos.DistanceFrom(prevPos)

		// Angle needs at least 3 points (i-1, i, i+1)
		if i+1 < n {
			nextPos := md.AimPoints[i+1].Pos
			angleDeg := getDirAngle(prevPos, currPos, nextPos)
			// Normalize angle to [-180, 180] - getDirAngle should handle this
			angles[i] = angleDeg
		} else {
			// Last point has no next
			angles[i] = 0.0
		}
	}

	md.Angles = angles
	md.Distances = distances
}

// populateReadingPoints constructs the ReadingPoint slice from AimPoints,
// Angles, Distances, and map properties (AR).
func populateReadingPoints(md *MapData) {
	ar := md.Map.ApproachRate
	preemptMs := arToMs(ar)

	var readingPoints []ReadingPoint
	for i, ap := range md.AimPoints {
		preemptTime := ap.Time - int(preemptMs)
		fadeInTime := preemptTime
		fadeOutTime := ap.Time

		// Угол для точки i находится в md.Angles[i-1], если 0 < i < len(md.AimPoints)-1
		angle := 0.0
		if i > 0 && i < len(md.AimPoints)-1 && i-1 < len(md.Angles) {
			angle = md.Angles[i-1]
		}

		readingPoints = append(readingPoints, ReadingPoint{
			Index:   i,
			Time:    ap.Time,
			Preempt: preemptTime,
			FadeIn:  fadeInTime,
			FadeOut: fadeOutTime,
			Angle:   angle,
			Pos:     ap.Pos,
			Dist:    md.Distances[i],
		})
	}

	md.ReadingPoints = readingPoints
}
