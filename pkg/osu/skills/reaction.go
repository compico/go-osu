package skills

import (
	"math"

	"github.com/compico/go-osu/pkg/osu"
)

// reactionConstA, reactionConstB precompute the two magic constants from
// react2Skill in reaction.cpp:
//
//	a = pow(2, log(78608/15625)/log(34/25)) * pow(125, log(68/25)/log(34/25))
//	b = log(2) / (log(2) - 2*log(5) + log(17))
//
// See https://www.desmos.com/calculator/lg2jqyesnu
var (
	reactionConstA = math.Pow(2.0, math.Log(78608.0/15625.0)/math.Log(34.0/25.0)) *
		math.Pow(125.0, math.Log(68.0/25.0)/math.Log(34.0/25.0))
	reactionConstB = math.Log(2.0) / (math.Log(2.0) - 2.0*math.Log(5.0) + math.Log(17.0))
)

// react2Skill ports reaction.cpp's react2Skill.
func react2Skill(timeToReact float64) float64 {
	return reactionConstA / math.Pow(timeToReact, reactionConstB)
}

// patternReq ports reaction.cpp's PatternReq.
func patternReq(p1, p2, p3 TargetPoint, csPx float64) float64 {
	dist := p1.Pos.DistanceFrom(p2.Pos) + p2.Pos.DistanceFrom(p3.Pos)
	angle := getAngle(p1.Pos, p2.Pos, p3.Pos)

	t := math.Abs(p3.Time - p1.Time)
	if t < 16 { // 16ms @ 60 FPS
		t = 16
	}

	// 2 * csPx = 1 diameter of CS since CS here is being calculated in terms of radius.
	return t / ((dist / (2 * csPx)) * ((math.Pi - angle) / math.Pi))
}

// pattern2Reaction ports reaction.cpp's Pattern2Reaction.
// See https://www.desmos.com/calculator/k9r2uipjfq
func pattern2Reaction(p1, p2, p3 TargetPoint, arMs, csPx float64, vars *Vars) float64 {
	damping := vars.Get("Reaction", "PatternDamping")
	patReq := patternReq(p1, p2, p3, csPx)

	return arMs - arMs*math.Exp(-damping*patReq)
}

// getReactionSkillAt ports reaction.cpp's getReactionSkillAt.
func getReactionSkillAt(targetPoints []TargetPoint, targetPoint TargetPoint, hitobjects []osu.HitObject, ar, cs float64, hidden bool, vars *Vars) float64 {
	fadeInReactReq := vars.Get("Reaction", "FadeinPercent")
	timeToReact := 0.0
	index := findTimingAt(targetPoints, targetPoint.Time)

	debugf("[REACTION] time=%v index=%v ar=%v cs=%v hidden=%d\n", targetPoint.Time, index, ar, cs, map[bool]int{false: 0, true: 1}[hidden])

	if index >= len(targetPoints)-2 {
		timeToReact = float64(arToMS(ar))
		debugf("[REACTION] branch=tail timeToReact=%v\n", timeToReact)
	} else if index < 3 {
		visibilityTimes := getVisibilityTimes(hitobjects[0], ar, hidden, fadeInReactReq, 1.0)
		timeToReact = float64(hitobjects[0].Time - visibilityTimes.first)
		debugf("[REACTION] branch=head visFirst=%v timeToReact=%v\n", visibilityTimes.first, timeToReact)
	} else {
		t1 := targetPoints[index]
		t2 := targetPoints[index+1]
		t3 := targetPoints[index+2]

		timeSinceStart := 0
		if targetPoint.Press {
			timeSinceStart = absInt(int(targetPoint.Time) - hitobjects[targetPoint.Key].Time)
		}

		visibilityTimes := getVisibilityTimes(hitobjects[0], ar, hidden, fadeInReactReq, 1.0)
		actualArTime := (hitobjects[0].Time - visibilityTimes.first) + timeSinceStart

		result := pattern2Reaction(t1, t2, t3, float64(actualArTime), float64(cs2px(cs)), vars)
		timeToReact = math.Sqrt(timeToReact*timeToReact + result*result)

		debugf("[REACTION] branch=mid timeSinceStart=%v actualArTime=%v cs2px=%v result=%v timeToReact=%v\n",
			timeSinceStart, actualArTime, cs2px(cs), result, timeToReact)
	}

	verScale := vars.Get("Reaction", "VerScale")
	curveExp := vars.Get("Reaction", "CurveExp")
	skillVal := verScale * math.Pow(react2Skill(timeToReact), curveExp)

	debugf("[REACTION] react2Skill=%v verScale=%v curveExp=%v skillVal=%v\n", react2Skill(timeToReact), verScale, curveExp, skillVal)

	return skillVal
}

// CalculateReaction ports reaction.cpp's CalculateReaction.
// Requires md.TargetPoints (populated by gatherTargetPoints).
func CalculateReaction(md *MapData, vars *Vars, hidden bool) {
	mx := 0.0
	avg := 0.0
	weight := vars.Get("Reaction", "AvgWeighting")

	for _, tick := range md.TargetPoints {
		val := getReactionSkillAt(md.TargetPoints, tick, md.Map.HitObjects, md.Map.ApproachRate, md.Map.CircleSize, hidden, vars)

		if val > mx {
			mx = val
		}
		if val > mx/2.0 {
			avg = weight*val + (1-weight)*avg
		}
	}

	md.Skills.Reaction = (mx + avg) / 2.0
}
