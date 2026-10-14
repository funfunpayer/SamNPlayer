package player

import (
	"context"
	"testing"
	"time"
)

func (d *recordingDevice) lastVibration() float64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.vibrations) == 0 {
		return -1
	}
	return d.vibrations[len(d.vibrations)-1]
}

func (d *recordingDevice) writes() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.vibrations) + len(d.suctions)
}

// Pause holds both channels at 0, no ramp step reaches the device while
// paused, and Resume gives the level back and lets the session finish.
func TestPauseHoldsOutputAndResumeContinues(t *testing.T) {
	dev := &recordingDevice{}
	control := NewTrainingControl()
	opts := TrainingOptions{Cycles: 1, PeakIntensity: 0.8, RampUpMs: 400, HoldMs: 200, RestMs: 100,
		Channel: ChannelVibration, Technique: TechniqueStopStart}
	done := make(chan error, 1)
	start := time.Now()
	go func() { done <- RunTrainingWithControl(context.Background(), dev, opts, control, nil) }()

	time.Sleep(200 * time.Millisecond) // mid ramp
	control.Pause()
	if !control.Paused() {
		t.Fatal("Paused() = false after Pause")
	}
	before := dev.lastVibration()
	if before != 0 {
		t.Fatalf("pause must zero the output at once, last vibration = %v", before)
	}
	if dev.lastSuction() != 0 {
		t.Fatalf("pause must zero suction too, got %v", dev.lastSuction())
	}
	n := dev.writes()
	time.Sleep(500 * time.Millisecond) // longer than the rest of the session
	if dev.writes() != n {
		t.Fatalf("device written while paused: %d → %d writes", n, dev.writes())
	}
	select {
	case err := <-done:
		t.Fatalf("session ended while paused (err=%v): the clock must stand still", err)
	default:
	}

	control.Resume()
	if control.Paused() {
		t.Fatal("Paused() = true after Resume")
	}
	if v := dev.lastVibration(); v <= 0 {
		t.Fatalf("resume must restore the level, got %v", v)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("session did not finish after Resume")
	}
	if took := time.Since(start); took < 1100*time.Millisecond {
		t.Fatalf("session took %v: the paused time must not count", took)
	}
}

// Ending the session while paused stops at once.
func TestCancelWhilePausedEndsSession(t *testing.T) {
	dev := &recordingDevice{}
	control := NewTrainingControl()
	opts := TrainingOptions{Cycles: 3, PeakIntensity: 0.5, RampUpMs: 1000, HoldMs: 1000, RestMs: 1000,
		Channel: ChannelBoth, Technique: TechniqueStopStart}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- RunTrainingWithControl(ctx, dev, opts, control, nil) }()
	time.Sleep(150 * time.Millisecond)
	control.Pause()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancel while paused did not end the session")
	}
}

// Pause and Resume twice in a row are no-ops; a nil control is safe.
func TestPauseResumeIdempotent(t *testing.T) {
	control := NewTrainingControl()
	control.Resume()
	control.Pause()
	control.Pause()
	control.Resume()
	control.Resume()
	if control.Paused() {
		t.Fatal("still paused")
	}
	var none *TrainingControl
	none.Pause()
	none.Resume()
	if none.Paused() {
		t.Fatal("nil control paused")
	}
}
