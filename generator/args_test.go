package generator

import (
	"strings"
	"testing"
)

func TestGeneratorOptionsReachPython(t *testing.T) {
	cases := []struct {
		name string
		opts Options
		want []string
	}{
		{"Standard ohne Extras", Options{}, nil},
		{"Auto-Retry", Options{AutoRetry: true}, []string{"--auto-retry"}},
		{"waagerechte Achse", Options{Axis: "x"}, []string{"--axis", "x"}},
		{"Geschwindigkeitsgrenze", Options{MaxSpeed: 400}, []string{"--max-speed", "400"}},
		{"adaptive Keyframes", Options{AdaptiveKeyframeError: 6}, []string{"--adaptive-keyframes", "6"}},
		{"Region pro Szene", Options{PerSceneROI: true}, []string{"--per-scene-roi"}},
		{"Profil tj", Options{Profile: "tj"}, []string{"--profile", "tj"}},
		{"Profil tf", Options{Profile: "tf"}, []string{"--profile", "tf"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := BuildArgsForTest(tc.opts)
			joined := strings.Join(args, " ")
			for _, w := range tc.want {
				if !strings.Contains(joined, w) {
					t.Errorf("Argument %q fehlt in: %s", w, joined)
				}
			}
			if tc.want == nil {
				for _, flag := range []string{"--auto-retry", "--axis", "--max-speed",
					"--adaptive-keyframes", "--per-scene-roi"} {
					if strings.Contains(joined, flag) {
						t.Errorf("unerwartetes Argument %q bei Standardoptionen: %s", flag, joined)
					}
				}
			}
		})
	}
}
