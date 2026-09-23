//go:build cgo && opencv

package trackcv

import (
	"math"
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
	// padded local-neighborhood search around it.
	driftedBox := Rect{X: 0, Y: 0, W: 20, H: 20}
	score, ok := m.matchesOriginal(gray, driftedBox)
	if ok && score >= m.minScore {
		t.Errorf("drifted-away box scored %.3f (ok=%v), wanted below minScore (%.3f) - the gate would wrongly accept it into memory",
			score, ok, m.minScore)
	}
}

// Regression: an earlier version of matchesOriginal sized its search
// window off the CURRENT box's own dimensions. On the real clip that
// motivated this fix, CSRT's box didn't just drift position - it shrank
// from its original ~171x216 down to ~50x65 over ~175s (its own,
// independent scale-adaptation drift). Once the box got small enough,
// a box-sized search window became SMALLER than templates[0] itself,
// so the size guard silently returned ok=false ("nothing to compare")
// for essentially every check from then on - defeating this whole gate
// exactly when it mattered most, without ever looking like an error
// (score stayed a quiet 0.000, easy to miss without exact instrumentation).
func TestAppearanceMemoryMatchesOriginalNotDefeatedByShrunkBox(t *testing.T) {
	const cx, cy, radius = 160, 120, 35
	roi := Rect{X: cx - radius, Y: cy - radius, W: 2 * radius, H: 2 * radius} // 70x70
	gray := grayFrameFromVideo(t, func(buf []byte) {
		fillCircle(buf, testW, testH, cx, cy, radius, 250, 250, 250)
	})
	defer gray.Close()

	m := newAppearanceMemory()
	defer m.close()
	m.remember(gray, roi)

	// A much smaller box (as if CSRT's scale estimate had collapsed),
	// still centered on the real circle - must still be recognized.
	shrunkOnTarget := Rect{X: cx - 8, Y: cy - 8, W: 16, H: 16}
	if score, ok := m.matchesOriginal(gray, shrunkOnTarget); !ok || score < m.minScore {
		t.Errorf("shrunk-but-still-on-target box scored %.3f (ok=%v), want >= minScore (%.3f) - "+
			"a box-sized search window would have missed this", score, ok, m.minScore)
	}

	// Same tiny size, but drifted away too - must still be rejected (the
	// fix must not have become so generous it stops catching real drift).
	shrunkAndDrifted := Rect{X: 2, Y: 2, W: 16, H: 16}
	if score, ok := m.matchesOriginal(gray, shrunkAndDrifted); ok && score >= m.minScore {
		t.Errorf("shrunk AND drifted box scored %.3f (ok=%v), want below minScore (%.3f)",
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

// TrackROI's periodic check (track.go) is a two-step active correction:
// (1) matchesOriginal flags that the current box has drifted off target,
// then (2) reacquire() is asked to find something better right then,
// instead of waiting for CSRT to eventually report an outright loss
// (which gradual drift, by construction, never does). This test proves
// step 2 actually works once step 1 has fired: with the true target
// still present elsewhere in the same frame, reacquire() must find it.
func TestAppearanceMemoryActiveCorrectionRecoversTrueTarget(t *testing.T) {
	const cx, cy, radius = 160, 120, 35
	roi := Rect{X: cx - radius, Y: cy - radius, W: 2 * radius, H: 2 * radius}
	gray := grayFrameFromVideo(t, func(buf []byte) {
		fillCircle(buf, testW, testH, cx, cy, radius, 250, 250, 250)
	})
	defer gray.Close()

	m := newAppearanceMemory()
	defer m.close()
	m.remember(gray, roi)

	// Step 1: a box that has wandered off target, as gradual drift would
	// produce - CSRT itself would still call this an "ok=true" update.
	driftedBox := Rect{X: 0, Y: 0, W: 20, H: 20}
	if score, known := m.matchesOriginal(gray, driftedBox); known && score >= m.minScore {
		t.Fatalf("test setup: drifted box unexpectedly matched (score=%.3f)", score)
	}

	// Step 2: the correction itself - the true target is still right
	// there in the frame, reacquire() must land back on it.
	found, reacquired := m.reacquire(gray)
	if !reacquired {
		t.Fatal("reacquire() failed to recover the true target still present in the frame")
	}
	foundCx, foundCy := found.X+found.W/2, found.Y+found.H/2
	dist := math.Hypot(float64(foundCx-cx), float64(foundCy-cy))
	if dist > radius {
		t.Errorf("reacquire() landed %.1fpx from the true circle center, want within its radius (%d)", dist, radius)
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
