package sam

import (
	"testing"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

func TestPlaybackFramesUsesSAMIntensity(t *testing.T) {
	fs, err := funscript.Parse([]byte(`{"actions":[
		{"at":0,"pos":20},{"at":500,"pos":20},
		{"at":600,"pos":90},{"at":700,"pos":90},{"at":900,"pos":90},
		{"at":1000,"pos":20},{"at":1500,"pos":20}
	],"metadata":{
		"profile":"tj",
		"device_recipe":{"sync":"suction_position","min_suction":0.2,
			"tick_ms":50,"max_speed":0.5,"smoothing":0.22,"contact_vibration":true},
		"tracking_gaps":[{"start_ms":650,"end_ms":850}]
	}}`))
	if err != nil {
		t.Fatal(err)
	}
	opts := funscript.RecipeFor("tj")
	opts.ContactVibration = true
	opts.ContactVibrationEnvelope = -1
	opts.Smoothing = 0
	opts.TrackingGaps = fs.Metadata.TrackingGaps

	frames := PlaybackFramesFromFunscript(fs, opts)
	if len(frames) == 0 {
		t.Fatal("keine Frames")
	}
	var vibFar, vibContact, vibGap float64
	for _, f := range frames {
		if f.At >= 400 && f.At <= 500 && f.Vibration > vibFar {
			vibFar = f.Vibration
		}
		if f.At >= 600 && f.At <= 640 && f.Vibration > vibContact {
			vibContact = f.Vibration
		}
		if f.At >= 700 && f.At <= 800 && f.Vibration > vibGap {
			vibGap = f.Vibration
		}
	}
	if vibFar > 0.001 {
		t.Errorf("fern: vib=%.3f", vibFar)
	}
	if vibContact < 0.5 {
		t.Errorf("Kontakt über SAM-Intensity: vib=%.3f", vibContact)
	}
	if vibGap > 0.001 {
		t.Errorf("Gap muss Vib 0 halten: vib=%.3f", vibGap)
	}
}

func TestPlaybackFramesFallsBackWithoutContact(t *testing.T) {
	fs, err := funscript.Parse([]byte(`{"actions":[
		{"at":0,"pos":20},{"at":500,"pos":90},{"at":1000,"pos":20}
	],"metadata":{"profile":"tj"}}`))
	if err != nil {
		t.Fatal(err)
	}
	opts := funscript.RecipeFor("tj")
	opts.Smoothing = 0
	frames := PlaybackFramesFromFunscript(fs, opts)
	for _, f := range frames {
		if f.Vibration > 0.001 {
			t.Fatalf("ohne Contact sollte Vib 0 sein, ist %.3f", f.Vibration)
		}
		if f.Suction < 0.1 {
			t.Fatalf("Sog sollte Positionsignal folgen")
		}
	}
}
