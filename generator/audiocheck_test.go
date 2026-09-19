package generator

import (
	"math"
	"testing"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

func sine(freqHz, durationS, sampleRate float64) []float64 {
	n := int(durationS * sampleRate)
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		out[i] = math.Sin(2 * math.Pi * freqHz * float64(i) / sampleRate)
	}
	return out
}

func TestDominantFrequencyHz(t *testing.T) {
	sr := 100.0
	sig := sine(1.5, 20, sr)
	freq := dominantFrequencyHz(sig, sr, audioMinTempoHz, audioMaxTempoHz)
	if freq == nil || math.Abs(*freq-1.5) > 0.1 {
		t.Fatalf("expected ~1.5Hz, got %v", freq)
	}
	if dominantFrequencyHz([]float64{1, 2, 3}, sr, audioMinTempoHz, audioMaxTempoHz) != nil {
		t.Fatal("short signal must yield nil")
	}
}

func TestEstimateScriptTempoHz(t *testing.T) {
	actions := make([]funscript.Action, 0, 500)
	for at := int64(0); at < 20000; at += 40 {
		pos := 50 + 45*math.Sin(2*math.Pi*1.0*float64(at)/1000.0)
		actions = append(actions, funscript.Action{At: at, Pos: int(math.Round(pos))})
	}
	hz := estimateScriptTempoHz(actions)
	if hz == nil || math.Abs(*hz-1.0) > 0.15 {
		t.Fatalf("expected ~1.0Hz, got %v", hz)
	}
	if estimateScriptTempoHz(actions[:3]) != nil {
		t.Fatal("too few actions must yield nil")
	}
}

func TestCompareTempo(t *testing.T) {
	one, two, off := 1.0, 2.0, 3.7
	exact := compareTempo(&one, &one, 0.25, audioDefaultHarmonics)
	if exact == nil || !exact.matches || exact.harmonic != 1.0 {
		t.Fatalf("1:1: %+v", exact)
	}
	dbl := compareTempo(&one, &two, 0.25, audioDefaultHarmonics)
	if dbl == nil || !dbl.matches || dbl.harmonic != 0.5 {
		t.Fatalf("0.5x: %+v", dbl)
	}
	bad := compareTempo(&one, &off, 0.25, audioDefaultHarmonics)
	if bad == nil || bad.matches {
		t.Fatalf("outlier should not match: %+v", bad)
	}
	if compareTempo(nil, &one, 0.25, audioDefaultHarmonics) != nil {
		t.Fatal("nil script_hz")
	}
}

func TestAudioEnvelope(t *testing.T) {
	sr := 4000.0
	samples := make([]float64, int(30*sr))
	for i := range samples {
		env := 0.5 + 0.5*math.Sin(2*math.Pi*1.0*float64(i)/sr)
		// deterministic "noise" carrier
		carrier := math.Sin(2*math.Pi*200*float64(i)/sr) + 0.3*math.Sin(2*math.Pi*350*float64(i)/sr)
		samples[i] = env * carrier
	}
	env, rate := audioEnvelope(samples, sr, 50.0)
	if len(env) < 8 || rate != 20.0 {
		t.Fatalf("envelope len=%d rate=%v", len(env), rate)
	}
	hz := estimateAudioTempoHz(samples, sr, 50.0)
	if hz == nil || math.Abs(*hz-1.0) > 0.2 {
		t.Fatalf("expected ~1.0Hz audio tempo, got %v", hz)
	}
}

func TestCheckAudioTempoNoAudio(t *testing.T) {
	if CheckAudioTempo("/nonexistent/nope.mp4", []funscript.Action{{At: 0, Pos: 0}}) != nil {
		t.Fatal("missing video must return nil")
	}
}
