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
		"oral": SceneFeelOral, "Blowjob": SceneFeelOral, "mouth": SceneFeelOral, "lips": SceneFeelOral,
		"penetration": SceneFeelIntercourse,
		"titjob": SceneFeelBreast, "nipple": SceneFeelBreast, "nipples": SceneFeelBreast,
		"handjob": SceneFeelHand, "": SceneFeelNone, "dance": SceneFeelNone,
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
		t.Fatal("none should not be ok")
	}
}

func TestOralRaisesSuction(t *testing.T) {
	s := &Script{Actions: strokeActions()}
	o := DefaultMapOptions()
	o.SceneFeel = SceneFeelOral
	fr := s.ToIntensityCurve(o)
	_, suc := meanFrames(fr)
	base := s.ToIntensityCurve(DefaultMapOptions())
	_, baseSuc := meanFrames(base)
	if suc <= baseSuc {
		t.Fatalf("oral should raise mean suction: %v vs %v", suc, baseSuc)
	}
	o.SceneFeel = SceneFeelIntercourse
	fr2 := s.ToIntensityCurve(o)
	_, suc2 := meanFrames(fr2)
	if suc2 <= baseSuc {
		t.Fatalf("intercourse should raise mean suction: %v vs %v", suc2, baseSuc)
	}
}

func TestVibrationFollowsSuction(t *testing.T) {
	s := &Script{Actions: strokeActions()}
	o := DefaultMapOptions()
	o.SceneFeel = SceneFeelOral
	o.VibrationFollowsSuction = true
	fr := s.ToIntensityCurve(o)
	for i, f := range fr {
		if math.Abs(f.Vibration-f.Suction) > 1e-9 {
			t.Fatalf("frame %d: vib %v != suc %v", i, f.Vibration, f.Suction)
		}
	}
}

func TestBalanceExtremes(t *testing.T) {
	s := &Script{Actions: strokeActions()}
	o := DefaultMapOptions()
	o.Balance = 1
	fr := s.ToIntensityCurve(o)
	for _, f := range fr {
		if f.Vibration != 0 {
			t.Fatalf("balance +1 should zero vibration: %+v", f)
		}
	}
	o.Balance = -1
	fr = s.ToIntensityCurve(o)
	for _, f := range fr {
		if f.Suction != 0 {
			t.Fatalf("balance -1 should zero suction: %+v", f)
		}
	}
	o.Balance = 0.5
	fr = s.ToIntensityCurve(o)
	base := s.ToIntensityCurve(DefaultMapOptions())
	for i := range fr {
		if math.Abs(fr[i].Vibration-base[i].Vibration*0.5) > 1e-6 {
			t.Fatalf("balance 0.5 vib scale: %v vs %v", fr[i].Vibration, base[i].Vibration*0.5)
		}
	}
	o.Balance = math.NaN()
	fr = s.ToIntensityCurve(o)
	for i := range fr {
		if fr[i] != base[i] {
			t.Fatalf("NaN balance should stay neutral: %+v vs %+v", fr[i], base[i])
		}
	}
}

func TestFeelRecipeRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.funscript")
	s := &Script{Actions: strokeActions()}
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	if err := SaveFeelRecipe(path, SceneFeelOral, true, 0.4); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	o := MapOptionsFromScript(loaded)
	if o.SceneFeel != SceneFeelOral || !o.VibrationFollowsSuction || math.Abs(o.Balance-0.4) > 1e-9 {
		t.Fatalf("got %+v", o)
	}
	// clear
	if err := SaveFeelRecipe(path, "", false, 0); err != nil {
		t.Fatal(err)
	}
	loaded, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	o = MapOptionsFromScript(loaded)
	if o.SceneFeel != "" || o.VibrationFollowsSuction || o.Balance != 0 {
		t.Fatalf("cleared got %+v", o)
	}
	_ = json.Marshal
	_ = os.TempDir
}
