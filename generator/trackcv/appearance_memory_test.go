//go:build cgo && opencv

package trackcv

import (
	"path/filepath"
	"testing"
)

// grayFrameFromVideo writes a one-frame synthetic video and returns its
// single frame as a *Gray - the same VideoWriter/OpenVideo/ToGray path
// track_test.go's other tests use, just for a single still frame instead
// of a motion sequence.
func grayFrameFromVideo(t *testing.T, draw func(buf []byte)) *Gray {
	t.Helper()
	path := filepath.Join(t.TempDir(), "frame.mp4")
	w := OpenVideoWriter(path, testFPS, testW, testH)
	buf := textureBackground(11)
	draw(buf)
	// A couple of repeated frames - some codecs/readers are unreliable
	// reading back a single-frame file.
	w.WriteBGR(buf, testW, testH, testW*3)
	w.WriteBGR(buf, testW, testH, testW*3)
	w.Close()

	cap := OpenVideo(path)
	t.Cleanup(cap.Close)
	if !cap.IsOpened() || !cap.Read() {
		t.Fatalf("could not read back synthetic frame")
	}
	return cap.ToGray()
}

// Real-clip finding (docs/AGENT_COORD.md, 23 Sep "CSRT long-clip drift"):
// a slow, gradual tracker drift left the appearance-memory bank full of
// crops of the wrong region by the time the tracker genuinely lost the
// target - so reacquire() confidently matched onto that same wrong
// region instead of recovering the real one. matchesOriginal is the
// fix's core check: does the CURRENT tracked box still look like
// templates[0] (the original, trusted region)?
func TestAppearanceMemoryMatchesOriginalAcceptsTheOriginalRegion(t *testing.T) {
	const cx, cy, radius = 160, 120, 35
	roi := Rect{X: cx - radius, Y: cy - radius, W: 2 * radius, H: 2 * radius}
	gray := grayFrameFromVideo(t, func(buf []byte) {
		fillCircle(buf, testW, testH, cx, cy, radius, 250, 250, 250)
	})
	defer gray.Close()

	m := newAppearanceMemory()
	defer m.close()
	m.remember(gray, roi)

	score, ok := m.matchesOriginal(gray, roi)
	if !ok {
		t.Fatal("matchesOriginal reported no match for the exact original region")
	}
	if score < m.minScore {
		t.Errorf("score for the original region itself is %.3f, want >= minScore (%.3f)", score, m.minScore)
	}
}

func TestAppearanceMemoryMatchesOriginalRejectsDriftedRegion(t *testing.T) {
	const cx, cy, radius = 160, 120, 35
	roi := Rect{X: cx - radius, Y: cy - radius, W: 2 * radius, H: 2 * radius}
	gray := grayFrameFromVideo(t, func(buf []byte) {
		fillCircle(buf, testW, testH, cx, cy, radius, 250, 250, 250)
	})
	defer gray.Close()

	m := newAppearanceMemory()
	defer m.close()
	m.remember(gray, roi)

	// A small box in the far corner, over plain textured background -
	// clearly disjoint from the circle even after matchesOriginal's own
	// padded local-neighborhood search around it (padding scales with
	// box size, so this needs to stay well clear at 3x its own size too).
	driftedBox := Rect{X: 0, Y: 0, W: 20, H: 20}
	score, ok := m.matchesOriginal(gray, driftedBox)
	if ok && score >= m.minScore {
		t.Errorf("drifted-away box scored %.3f (ok=%v), wanted below minScore (%.3f) - the gate would wrongly accept it into memory",
			score, ok, m.minScore)
	}
}

func TestAppearanceMemoryMatchesOriginalNoTemplatesYetAllowsRemembering(t *testing.T) {
	m := newAppearanceMemory()
	defer m.close()
	gray := grayFrameFromVideo(t, func(buf []byte) {
		fillCircle(buf, testW, testH, 160, 120, 35, 250, 250, 250)
	})
	defer gray.Close()

	if score, ok := m.matchesOriginal(gray, Rect{X: 10, Y: 10, W: 20, H: 20}); ok {
		t.Errorf("expected ok=false with no templates yet, got ok=true score=%.3f", score)
	}
}

// End-to-end: TrackROI now seeds templates[0] from the true frame-0 ROI
// (not whatever the tracker believes ~1s in) and gates the periodic
// remember() call on matchesOriginal - a run on the existing
// scene-cut/camera-compensation synthetic videos must behave exactly as
// before (this is a defensive addition, not a behavior change on content
// that never drifts).
func TestTrackROIStillWorksWithGatedAppearanceMemory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "amem.mp4")
	writeMovingVideo(t, path, 4, 80, 1.0)

	roi := Rect{X: 160 - 35, Y: 120 - 40, W: 70, H: 80}
	result, err := TrackROI(path, roi, Options{AppearanceMemory: true})
	if err != nil {
		t.Fatalf("TrackROI: %v", err)
	}
	if result.Stats.ValidFrames < result.Stats.TotalFrames/2 {
		t.Errorf("too many frames lost with gated AppearanceMemory: valid=%d total=%d",
			result.Stats.ValidFrames, result.Stats.TotalFrames)
	}
}
