package funscript

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func strokeActions() []Action {
	var a []Action
	for i := 0; i <= 40; i++ {
		pos := 10
		if i%2 == 1 {
			pos = 90
		}
		a = append(a, Action{At: int64(i * 400), Pos: pos})
	}
	return a
}

func meanFrames(fr []Frame) (vib, suc float64) {
	for _, f := range fr {
		vib += f.Vibration
		suc += f.Suction
	}
	n := float64(len(fr))
	return vib / n, suc / n
}

func TestNormalizeSceneFeelAliases(t *testing.T) {
	cases := map[string]string{
		"oral": SceneFeelOral, "Blowjob": SceneFeelOral, "penetration": SceneFeelIntercourse,
		"titjob": SceneFeelBreast, "handjob": SceneFeelHand, "": SceneFeelNone, "dance": SceneFeelNone,
	}
	for in, want := range cases {
		if got := NormalizeSceneFeel(in); got != want {
			t.Errorf("NormalizeSceneFeel(%q) = %q, want %q", in, got, want)
		}
	}
}

// Without scene feel, follow and balance the mapping is exactly the old one.
func TestSceneFeelOffKeepsOldMapping(t *testing.T) {
	s := &Script{Actions: strokeActions()}
	base := s.ToIntensityCurve(DefaultMapOptions())
	o := DefaultMapOptions()
	o.SceneFeel, o.VibrationFollowsSuction, o.Balance = "", false, 0
	again := s.ToIntensityCurve(o)
	if len(base) != len(again) {
		t.Fatal("frame count changed")
	}
	for i := range base {
		if base[i] != again[i] {
			t.Fatalf("frame %d changed: %+v vs %+v", i, base[i], again[i])
		}
	}
}

func TestSceneSuctionShapes(t *testing.T) {
	// Oral keeps a seal even at the shallowest point.
	if v, _ := SceneSuction(SceneFeelOral, 0, 0); v < 0.29 {
		t.Fatalf("oral seal %v", v)
	}
	// Deeper pulls harder for every scene.
	for _, f := range SceneFeels {
		lo, _ := SceneSuction(f, 0.1, 0.5)
		hi, _ := SceneSuction(f, 0.9, 0.5)
		if hi <= lo {
			t.Errorf("%s: deeper should pull harder (%v ≤ %v)", f, hi, lo)
		}
	}
	// Hand follows speed more than depth; breast follows depth more than speed.
	hSpeed, _ := SceneSuction(SceneFeelHand, 0.5, 1)
	hDepth, _ := SceneSuction(SceneFeelHand, 1, 0.5)
	if hSpeed <= hDepth {
		t.Errorf("hand: speed should weigh more (%v ≤ %v)", hSpeed, hDepth)
	}
	bSpeed, _ := SceneSuction(SceneFeelBreast, 0.5, 1)
	bDepth, _ := SceneSuction(SceneFeelBreast, 1, 0.5)
	if bDepth <= bSpeed {
		t.Errorf("breast: depth should weigh more (%v ≤ %v)", bDepth, bSpeed)
	}
	if _, ok := SceneSuction(SceneFeelNone, 0.5, 0.5); ok {
		t.Error("none must not shape suction")
	}
}

func TestOralSceneRaisesSuction(t *testing.T) {
	s := &Script{Actions: strokeActions()}
	_, oldSuc := meanFrames(s.ToIntensityCurve(DefaultMapOptions()))
	o := DefaultMapOptions()
	o.SceneFeel = SceneFeelOral
	_, oralSuc := meanFrames(s.ToIntensityCurve(o))
	if oralSuc <= oldSuc {
		t.Fatalf("oral suction %v should be above position suction %v", oralSuc, oldSuc)
	}
}

func TestVibrationFollowsSuction(t *testing.T) {
	s := &Script{Actions: strokeActions()}
	o := DefaultMapOptions()
	o.SceneFeel = SceneFeelIntercourse
	o.VibrationFollowsSuction = true
	o.Smoothing = 0
	for _, f := range s.ToIntensityCurve(o) {
		if math.Abs(f.Vibration-f.Suction) > 1e-9 {
			t.Fatalf("vibration %v does not follow suction %v at %d", f.Vibration, f.Suction, f.At)
		}
	}
}

func TestBalance(t *testing.T) {
	s := &Script{Actions: strokeActions()}
	o := DefaultMapOptions()
	o.SceneFeel = SceneFeelOral
	o.VibrationFollowsSuction = true
	o.Balance = 1
	vib, suc := meanFrames(s.ToIntensityCurve(o))
	if vib > 1e-9 || suc <= 0 {
		t.Fatalf("balance +1: vib %v suc %v", vib, suc)
	}
	o.Balance = -1
	vib, suc = meanFrames(s.ToIntensityCurve(o))
	if suc > 1e-9 || vib <= 0 {
		t.Fatalf("balance −1: vib %v suc %v", vib, suc)
	}
	o.Balance = 0.5
	vHalf, sHalf := meanFrames(s.ToIntensityCurve(o))
	o.Balance = 0
	vZero, sZero := meanFrames(s.ToIntensityCurve(o))
	if math.Abs(vHalf-vZero*0.5) > 1e-6 || math.Abs(sHalf-sZero) > 1e-6 {
		t.Fatalf("balance 0.5: vib %v (want %v), suc %v (want %v)", vHalf, vZero*0.5, sHalf, sZero)
	}
	if v, su := BalanceScales(math.NaN()); v != 1 || su != 1 {
		t.Fatal("NaN balance must be neutral")
	}
}

func TestSaveFeelRecipeRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.funscript")
	raw, _ := json.Marshal(map[string]any{"actions": strokeActions(), "metadata": map[string]any{"creator": "x"}})
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SaveFeelRecipe(path, "blowjob", true, 0.4); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	o := MapOptionsFromScript(s)
	if o.SceneFeel != SceneFeelOral || !o.VibrationFollowsSuction || math.Abs(o.Balance-0.4) > 1e-9 {
		t.Fatalf("options %+v", o)
	}
	if s.Metadata.Creator != "x" {
		t.Fatal("other metadata lost")
	}
}
