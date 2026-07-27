package skills

import "math"

// Constants ported from reading.cpp.
const (
	readingDistanceInfluenceThreshold = 150.0
	readingMinAngleRelevancyTime      = 2000.0
	readingMaxAngleRelevancyTime      = 200.0
	readingWindowSize                 = 3000.0
	readingHiddenMultiplier           = 0.28
	readingFadeOutDurationMultiplier  = 0.3
	readingStrainDecayBase            = 0.8
	readingMinDeltaTime               = 15.0

	readingDensityDifficultyBase = 2.5
	readingDensityMultiplier     = 2.4
)

// CalculateReading ports reading.cpp's CalculateReading. It requires
// md.ReadingPoints to already be populated (by the aim/reading point
// preparation step — not yet ported; see the package doc comment on
// MapData) and writes the resulting strain curve and skill value back
// into md.
func CalculateReading(md *MapData, vars *Vars, hidden bool) {
	n := len(md.ReadingPoints)
	strain := 0.0

	for i := 0; i < n; i++ {
		current := &md.ReadingPoints[i]

		var previous *ReadingPoint
		if i-1 >= 0 {
			previous = &md.ReadingPoints[i-1]
		}

		var next *ReadingPoint
		interval := readingWindowSize
		if i+1 < n {
			next = &md.ReadingPoints[i+1]
			interval = math.Max(readingMinDeltaTime, float64(next.Time-current.Time))
		}

		diff := evaluateDifficultyOf(md.ReadingPoints, current, previous, next, md.Map.ApproachRate, hidden)
		decay := math.Pow(readingStrainDecayBase, interval/1000)

		strain *= decay
		strain += diff * (1 - decay)
		md.ReadingStrains = append(md.ReadingStrains, strain)
	}

	// Note: the original C++ also computed
	// beatmap.skills.reading = *max_element(readingStrains) here first —
	// but that value is immediately discarded and overwritten by the
	// weighted-value formula below, so it's dropped as dead code. (It was
	// also technically UB on an empty readingStrains, e.g. a 0- or 1-note
	// map, since std::max_element on an empty range can't be dereferenced.)
	topWeights := getPeakVals(md.ReadingStrains)

	md.Skills.Reading = getWeightedValue2(topWeights, vars.Get("Reading", "Weighting"))
	md.Skills.Reading = vars.Get("Reading", "TotalMult") * math.Pow(md.Skills.Reading, vars.Get("Reading", "TotalPow"))
}

// evaluateDifficultyOf ports reading.cpp's EvaluateDifficultyOf.
//
// Deviation from the original: the C++ has `if (&previous)` here, which —
// because &previous is the address of a reference parameter, never a nil
// check on the pointee — always evaluates true, regardless of whether a
// real previous point exists. That reads as a bug: the clearly intended
// check (matching the correctly-written `if (&previous == nullptr) return 0;`
// a few functions over, in calculateHiddenDifficulty) is "do we have a
// previous point". We implement the intended behavior (previous != nil)
// here. Replicating the literal C++ would also not be meaningfully
// possible in Go without dereferencing a nil pointer at i==0 and crashing.
// The only observable numeric difference from the original binary is for
// each map's very first reading point.
func evaluateDifficultyOf(points []ReadingPoint, current, previous, next *ReadingPoint, ar float64, hidden bool) float64 {
	interval := readingWindowSize
	if previous != nil {
		interval = math.Max(readingMinDeltaTime, float64(current.Time-previous.Time))
	}

	velocity := math.Max(1.0, current.Dist/interval)
	pastObjectDifficultyInfluence := getPastObjectDifficultyInfluence(points, current, ar)

	constantAngleNerfFactor := getConstantAngleNerfFactor(points, current)
	currentVisibleObjectDensity := retrieveCurrentVisibleObjectDensity(points, current, ar)
	noteDensityDifficulty := calculateDensityDifficulty(current, next, velocity, constantAngleNerfFactor, pastObjectDifficultyInfluence, currentVisibleObjectDensity)

	hiddenDifficulty := 0.0
	if hidden {
		hiddenDifficulty = calculateHiddenDifficulty(current, previous, ar, pastObjectDifficultyInfluence, currentVisibleObjectDensity, velocity, constantAngleNerfFactor)
	}

	readingDifficulty := norm(1.5, []float64{hiddenDifficulty, noteDensityDifficulty})

	// Having less time to process information is harder.
	readingDifficulty *= 1.0 / (1.0 - math.Pow(0.5, interval/1000))

	return readingDifficulty
}

// calculateDensityDifficulty ports reading.cpp's calculateDensityDifficulty.
func calculateDensityDifficulty(current, next *ReadingPoint, velocity, constantAngleNerfFactor, pastObjectDifficultyInfluence, currentVisibleObjectDensity float64) float64 {
	// Consider future densities too because it can make the path the
	// cursor takes less clear.
	futureObjectDifficultyInfluence := math.Sqrt(currentVisibleObjectDensity)

	if next != nil {
		// Reduce difficulty if movement to next object is small.
		futureObjectDifficultyInfluence *= smootherStep(next.Pos.DistanceFrom(current.Pos), 15, readingDistanceInfluenceThreshold)
	}

	// Value higher note densities exponentially.
	noteDensityDifficulty := math.Pow(pastObjectDifficultyInfluence+futureObjectDifficultyInfluence, 1.7) * 0.4 * constantAngleNerfFactor * velocity

	// Award only denser than average maps.
	noteDensityDifficulty = math.Max(0.0, noteDensityDifficulty-readingDensityDifficultyBase)

	// Apply a soft cap to general density reading to account for partial
	// memorization.
	noteDensityDifficulty = math.Pow(noteDensityDifficulty, 0.45) * readingDensityMultiplier

	return noteDensityDifficulty
}

// getConstantAngleNerfFactor ports reading.cpp's getConstantAngleNerfFactor.
// Returns a factor of how often the current object's angle has been
// repeated in a certain time frame. It does this by checking the
// difference in angle between current and past objects and sums them
// based on a range of similarity.
// https://www.desmos.com/calculator/eb057a4822
func getConstantAngleNerfFactor(points []ReadingPoint, current *ReadingPoint) float64 {
	constantAngleCount := 0.0
	currentTimeGap := 0.0

	loopObjPrev0 := current
	var loopObjPrev1, loopObjPrev2 *ReadingPoint

	for currentTimeGap < readingMinAngleRelevancyTime {
		loopID := loopObjPrev0.Index - 1
		if loopID < 0 {
			break
		}

		loopObj := &points[loopID]

		// Account less for objects that are close to the time limit.
		currentTimeGap = float64(current.Time - loopObj.Time)

		longIntervalFactor := 1 - reverseLerp(math.Max(currentTimeGap, 15), readingMaxAngleRelevancyTime, readingMinAngleRelevancyTime)

		if current.Angle != 0.0 && loopObjPrev1 != nil && loopObjPrev2 != nil {
			angleDifference := math.Abs(current.Angle - loopObj.Angle)

			angleDifferenceAlternating := math.Abs(loopObjPrev1.Angle-loopObj.Angle) + math.Abs(loopObjPrev2.Angle-loopObjPrev0.Angle)

			weight := 1.0
			// Be sure that one of the angles is very sharp, when other is wide.
			weight *= reverseLerp(math.Min(loopObj.Angle, loopObjPrev0.Angle), 20, 5)
			weight *= reverseLerp(math.Max(loopObj.Angle, loopObjPrev0.Angle), 60, 120)

			// Lerp between max angle difference and rescaled alternating
			// difference, with more harsh scaling compared to normal
			// difference.
			angleDifferenceAlternating = lerp(180, 0.1*angleDifferenceAlternating, weight)

			stackFactor := smootherStep(loopObj.Dist, 0, 50)
			constantAngleCount += math.Cos(3*math.Min(30, math.Min(angleDifference, angleDifferenceAlternating)*stackFactor)) * longIntervalFactor
		}

		loopObjPrev2 = loopObjPrev1
		loopObjPrev1 = loopObjPrev0
		loopObjPrev0 = loopObj
	}

	return clampVal(2.0/constantAngleCount, 0.2, 1.0)
}

// getPastObjectDifficultyInfluence ports reading.cpp's
// getPastObjectDifficultyInfluence.
func getPastObjectDifficultyInfluence(points []ReadingPoint, current *ReadingPoint, ar float64) float64 {
	pastObjectDifficultyInfluence := 0.0

	var previous *ReadingPoint
	for _, loopObj := range retrievePastVisibleObjects(points, current) {
		loopDifficulty := getOpacityAt(current, loopObj.Time, ar, false)

		// When aiming an object small distances mean previous objects may
		// be cheesed, so it doesn't matter whether they were arranged
		// confusingly.
		if previous != nil {
			loopDifficulty *= smootherStep(loopObj.Dist, 15, readingDistanceInfluenceThreshold)
		} else {
			loopDifficulty = 0
		}

		// Account less for objects close to the max reading window.
		timeBetweenCurrAndLoopObj := float64(current.Time - loopObj.Time)
		timeNerfFactor := getTimeNerfFactor(timeBetweenCurrAndLoopObj)

		loopDifficulty *= timeNerfFactor
		pastObjectDifficultyInfluence += loopDifficulty
		previous = loopObj
	}

	return pastObjectDifficultyInfluence
}

// getOpacityAt ports reading.cpp's GetOpacityAt.
func getOpacityAt(point *ReadingPoint, time int, ar float64, hidden bool) float64 {
	if time > point.Time {
		// Consider a ReadingPoint as being invisible when its start time
		// is passed. In reality the ReadingPoint will be visible beyond
		// its start time up until its hittable window has passed, but
		// this is an approximation and such a case is unlikely to be hit
		// where this function is used.
		return 0.0
	}

	fadeInStartTime := float64(point.Preempt)
	fadeInDuration := math.Min(float64(arToMs(ar)), 400)

	if hidden {
		fadeInDuration = 0.4 * float64(arToMs(ar))
		fadeOutStartTime := float64(point.FadeOut)
		fadeOutDuration := float64(point.Preempt) * readingFadeOutDurationMultiplier

		return math.Min(
			clampVal((float64(time)-fadeInStartTime)/fadeInDuration, 0.0, 1.0),
			1.0-clampVal((float64(time)-fadeOutStartTime)/fadeOutDuration, 0.0, 1.0),
		)
	}

	return clampVal((float64(time)-fadeInStartTime)/fadeInDuration, 0.0, 1.0)
}

// getTimeNerfFactor ports reading.cpp's getTimeNerfFactor: a nerfing factor
// for when objects are very distant in time, affecting reading less.
func getTimeNerfFactor(deltaTime float64) float64 {
	return clampVal(2.0-deltaTime/(readingWindowSize/2.0), 0.0, 1.0)
}

// retrievePastVisibleObjects ports reading.cpp's retrievePastVisibleObjects:
// a list of objects that are visible on screen at the point in time the
// current object becomes visible, oldest first.
func retrievePastVisibleObjects(points []ReadingPoint, current *ReadingPoint) []*ReadingPoint {
	var pastObjects []*ReadingPoint

	for i := current.Index; i > 0; i-- {
		prevPoint := &points[i-1]

		if current.Time-prevPoint.Time > 3000 ||
			prevPoint.Time < current.Preempt { // Current object not visible at the time object needs to be clicked.
			break
		}

		pastObjects = append(pastObjects, prevPoint)
	}

	for l, r := 0, len(pastObjects)-1; l < r; l, r = l+1, r-1 {
		pastObjects[l], pastObjects[r] = pastObjects[r], pastObjects[l]
	}

	return pastObjects
}

// retrieveCurrentVisibleObjectDensity ports reading.cpp's
// retrieveCurrentVisibleObjectDensity: the density of objects visible at
// the point in time the current object needs to be clicked, capped by the
// reading window.
func retrieveCurrentVisibleObjectDensity(points []ReadingPoint, current *ReadingPoint, ar float64) float64 {
	visibleObjectCount := 0.0

	for i := current.Index; i < len(points)-1; i++ {
		nextPoint := &points[i+1]
		timeDiff := float64(nextPoint.Time - current.Time)
		if timeDiff > readingWindowSize ||
			current.Time < nextPoint.Preempt { // Object not visible at the time current object needs to be clicked.
			break
		}

		timeNerfFactor := getTimeNerfFactor(timeDiff)

		visibleObjectCount += getOpacityAt(nextPoint, current.Time, ar, false) * timeNerfFactor
	}

	return visibleObjectCount
}

// calculateHiddenDifficulty ports reading.cpp's calculateHiddenDifficulty.
func calculateHiddenDifficulty(currObj, previous *ReadingPoint, ar, pastObjectDifficultyInfluence, currentVisibleObjectDensity, velocity, constantAngleNerfFactor float64) float64 {
	if previous == nil {
		return 0
	}

	// Higher preempt means that time spent invisible is higher too, we
	// want to reward that.
	//
	// Note: the original computes
	//   preemptFactor = pow(currObj.time - currObj.preempt, 2.2) * 0.01
	// and then immediately overwrites it with 1 on the next line, making
	// that computation dead code. It's dropped here.
	preemptFactor := 1.0

	// Account for both past and current densities.
	densityFactor := math.Pow(currentVisibleObjectDensity+pastObjectDifficultyInfluence, 3.3) * 3

	hiddenDifficulty := (preemptFactor + densityFactor) * constantAngleNerfFactor * velocity * 0.01

	// Apply a soft cap to general HD reading to account for partial
	// memorization.
	hiddenDifficulty = math.Pow(hiddenDifficulty, 0.4) * readingHiddenMultiplier

	// Buff perfect stacks only if current note is completely invisible at
	// the time you click the previous note.
	if currObj.Dist == 0.0 && getOpacityAt(currObj, previous.Time, ar, true) == 0 && previous.Time > currObj.Preempt {
		// Perfect stacks are harder the less time between notes.
		hiddenDifficulty += readingHiddenMultiplier * 2500 / math.Pow(float64(currObj.Time), 1.5)
	}

	return hiddenDifficulty
}
