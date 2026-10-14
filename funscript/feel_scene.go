package funscript

import "strings"

// Scene feel (Owner 14 Oct): how the Sam Neo 2's suction follows a stroke
// depends on the scene. Oral wants a steady seal that pulls deeper with each
// stroke; intercourse follows the rhythm; breast scenes stay softer and lean
// on depth; hand scenes grip with speed. Vibration can follow the same
// pattern as suction, and a balance moves the weight between the two.
//
// All of this is additive: an empty SceneFeel, VibrationFollowsSuction off
// and Balance 0 give exactly the old mapping.

const (
	SceneFeelNone        = ""
	SceneFeelOral        = "oral"
	SceneFeelIntercourse = "intercourse"
	SceneFeelBreast      = "breast"
	SceneFeelHand        = "hand"
)

// SceneFeels lists the scene feels in UI order.
var SceneFeels = []string{SceneFeelOral, SceneFeelIntercourse, SceneFeelBreast, SceneFeelHand}

// NormalizeSceneFeel maps names (also the AI scene types blowjob / titjob /
// handjob / penetration, and contact-mark classes mouth / nipples) to a
// scene feel; unknown → none.
func NormalizeSceneFeel(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "oral", "blowjob", "bj", "mouth", "lips":
		return SceneFeelOral
	case "intercourse", "penetration", "sex":
		return SceneFeelIntercourse
	case "breast", "titjob", "tj", "boobjob", "nipple", "nipples":
		return SceneFeelBreast
	case "hand", "handjob", "hj":
		return SceneFeelHand
	}
	return SceneFeelNone
}

// sceneShape: suction = Floor + Range*(DepthW*depth + SpeedW*intensity).
type sceneShape struct{ Floor, Range, DepthW, SpeedW float64 }

var sceneShapes = map[string]sceneShape{
	// Oral: a steady seal (floor), deeper strokes pull harder, each stroke
	// adds a pulse.
	SceneFeelOral: {Floor: 0.30, Range: 0.70, DepthW: 0.65, SpeedW: 0.35},
	// Intercourse: rhythm first, depth second.
	SceneFeelIntercourse: {Floor: 0.15, Range: 0.85, DepthW: 0.45, SpeedW: 0.55},
	// Breast: softer, mostly depth (the squeeze), little speed.
	SceneFeelBreast: {Floor: 0.20, Range: 0.55, DepthW: 0.70, SpeedW: 0.30},
	// Hand: the grip follows speed.
	SceneFeelHand: {Floor: 0.10, Range: 0.70, DepthW: 0.25, SpeedW: 0.75},
}

// SceneSuction is the suction level (0–1) for a scene feel at a depth
// (pos / 100) and a stroke intensity (speed / MaxSpeed). ok=false for none.
func SceneSuction(feel string, depth, intensity float64) (float64, bool) {
	s, ok := sceneShapes[feel]
	if !ok {
		return 0, false
	}
	return clamp01(s.Floor + s.Range*(s.DepthW*clamp01(depth)+s.SpeedW*clamp01(intensity))), true
}

// BalanceScales turns a balance (−1 … 1; >0 more suction) into multipliers
// for the two channels. The favoured channel stays at 1; the other is reduced.
func BalanceScales(balance float64) (vibScale, sucScale float64) {
	b := ClampBalance(balance)
	if b > 0 {
		return 1 - b, 1
	}
	if b < 0 {
		return 1, 1 + b
	}
	return 1, 1
}

// ClampBalance keeps balance in [−1, 1]; NaN becomes 0.
func ClampBalance(b float64) float64 {
	if b != b { // NaN
		return 0
	}
	if b < -1 {
		return -1
	}
	if b > 1 {
		return 1
	}
	return b
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
