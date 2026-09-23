package funscript

import "testing"

// Feel-decouple Stage A: shallow stroke but tip over contact mark → vib on.
func TestSpatialFeelBoostsShallowStroke(t *testing.T) {
	s := &Script{
		Actions: []Action{
			{At: 0, Pos: 20},
			{At: 1000, Pos: 30}, // shallow — below default contact depth slice
			{At: 2000, Pos: 20},
		},
	}
	s.Metadata.ContactMarks = &ContactMarks{
		Primary: &ContactMarkBox{X: 40, Y: 40, W: 40, H: 40},
	}
	s.Metadata.Trajectory = &TrajectoryData{
		Tip: []TrajectoryPoint{
			{AtMs: 0, X: 0, Y: 0},
			{AtMs: 1000, X: 60, Y: 60},
			{AtMs: 2000, X: 0, Y: 0},
		},
	}
	opts := DefaultMapOptions()
	opts.TickMs = 50
	opts.ContactVibration = true
	opts.ContactVibrationSpan = 0.75
	opts.ContactVibrationEnvelope = -1
	opts.Sync = SyncIndependent

	frames := s.ToIntensityCurve(opts)
	var mid, edge float64
	for _, f := range frames {
		if f.At == 1000 {
			mid = f.Vibration
		}
		if f.At == 0 {
			edge = f.Vibration
		}
	}
	if mid < 0.5 {
		t.Fatalf("expected spatial vib at tip-on-mark: mid=%v (edge=%v)", mid, edge)
	}
	if edge > mid {
		t.Fatalf("edge vib should not exceed mid: edge=%v mid=%v", edge, mid)
	}
}
