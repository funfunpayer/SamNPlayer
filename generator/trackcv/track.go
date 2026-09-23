//go:build cgo && opencv

// Package trackcv (track.go) ist eine Go-native Entsprechung von
// generate_funscript.py's track_roi() (CSRT + Szenenschnitt-Erkennung +
// Kamerakompensation + Erscheinungsgedächtnis) - derselbe Algorithmus,
// direkt gegen dieselbe OpenCV-C++-Bibliothek, ohne Python-Laufzeit.
//
// GEMESSEN (nicht nur angenommen): ein Feasibility-Test (CSRT-Tracker
// allein, dasselbe Testvideo/ROI, Python vs. dieser Go-Wrapper) ergab
// r=0.9996 Korrelation und max. 3px Abweichung bei ~15% weniger Laufzeit
// (kein Python-Interpreter-Overhead pro Frame, dieselbe CSRT-Rechenarbeit
// in der C++-Bibliothek). Siehe docs/NEXT.md für die Zahlen.
//
// Bewusst (noch) NICHT der Standardpfad: generator.go ruft für die
// Erzeugung weiterhin ausschließlich generate_funscript.py auf (Glättung,
// Keyframe-Extraktion, Quality Doctor, Funscript-Schreiben laufen dort,
// nicht hier) - dieses Paket deckt bisher nur den rechenintensivsten
// Teilschritt (das Tracking selbst) ab, nicht die gesamte Pipeline. Wie
// (oder ob) es eingebunden wird, ist eine offene Architekturfrage, siehe
// docs/NEXT.md.
package trackcv

import (
	"errors"
	"math"
)

// Options steuert TrackROI - entspricht track_roi()'s Parametern.
type Options struct {
	MaxFrames          int // 0 = unbegrenzt
	StartTimeSec       float64
	CameraCompensation bool
	SceneCutDetection  bool
	AppearanceMemory   bool
	Axis               string // "auto" (Standard), "x" oder "y"
	// Cancel, if non-nil, is checked each frame; return true to abort.
	// Used by GenerateWithContext so Abbrechen stops native CSRT too.
	Cancel func() bool
	// FixedB keeps ROI B at the initial box (static Tf/Tj contact target).
	FixedB bool
	// OnProgress is called ~every 1% of frames (done, total). total may be 0
	// when the container reports no frame count — GUI then shows indeterminate.
	OnProgress func(done, total int)
	// CaptureTrajectory opts into keeping raw per-frame tip/partner center
	// points on Result.TrajectoryA/B (MT-Debug overlay data). Off by
	// default: zero extra allocation/behavior when false.
	CaptureTrajectory bool
	// RhythmGrid takes the stroke signal from the most rhythmic optical-flow
	// cell near the box instead of the box's own motion (see
	// rhythm_grid.go). CSRT still tracks - it anchors the search and gives
	// the sign - so trajectory/stats are unchanged; only Positions differ.
	// Opt-in: costs a Farneback flow per frame (measured ~+18% runtime).
	RhythmGrid bool
}

// Stats entspricht dem stats-Teil, den backends.py's Vertrag verlangt
// (siehe generator/backends.py) - dieselben Feldnamen wie die
// Python-Backends, damit ein künftiger gemeinsamer Aufrufer beide Wege
// gleich behandeln kann.
type Stats struct {
	TrackerLostFrames int
	CameraFramesLost  int
	TotalFrames       int
	VerticalRange     float64
	HorizontalRange   float64
	// Observation contract (FINDINGS F-004): honest missing-data signal.
	ValidFrames int     // frames with a successful tracker update
	Confidence  float64 // 0..1 ≈ valid/total after frame 0
	Reason      string  // empty if ok; else e.g. "tracker_lost_heavy"
}

// Result ist die Go-Entsprechung von track_roi()'s Rückgabe.
type Result struct {
	TimestampsMs []int
	Positions    []float64 // je nach Options.Axis: x- oder y-Positionen
	Width        int
	Height       int
	SceneCuts    []int
	Stats        Stats
	Canceled     bool // true if Options.Cancel aborted the loop
	// LostFlags is set by TrackTwoPoints (per-frame: either tracker lost).
	LostFlags []bool
	// TrajectoryA/B: per-frame tip/partner center points in video-pixel
	// space, same length/order as TimestampsMs - only populated when
	// Options.CaptureTrajectory is true (MT-Debug overlay data). TrackROI
	// only ever fills TrajectoryA (single ROI, no partner).
	TrajectoryA []Point
	TrajectoryB []Point
}

// Point is a video-pixel-space (x,y) sample - MT-Debug trajectory capture.
type Point struct{ X, Y float64 }

func boxCenter(r Rect) Point {
	return Point{X: float64(r.X) + float64(r.W)/2, Y: float64(r.Y) + float64(r.H)/2}
}

// ErrCanceled is returned when Options.Cancel aborts the tracking loop.
var ErrCanceled = errors.New("tracking abgebrochen")

const (
	sceneCutHistThreshold = 0.5
	sceneCutDiffThreshold = 12.0
	rememberEveryNFrames  = 25
)

// TrackROI verfolgt roi=(x,y,w,h) durchs Video unter videoPath. Siehe
// Paketkommentar für die Entsprechung zu track_roi() in
// generate_funscript.py - dieselbe Logik, Frame für Frame nachgebildet.
func TrackROI(videoPath string, roi Rect, opts Options) (Result, error) {
	cap := OpenVideo(videoPath)
	defer cap.Close()
	if !cap.IsOpened() {
		return Result{}, &trackError{"Video konnte nicht geöffnet werden: " + videoPath}
	}

	fps := cap.Get(CapPropFPS)
	if fps <= 0 {
		fps = 30.0
	}
	width := int(cap.Get(CapPropFrameWidth))
	height := int(cap.Get(CapPropFrameHeight))

	startFrame := 0
	if opts.StartTimeSec > 0 {
		startFrame = int(opts.StartTimeSec*fps + 0.5)
		if startFrame > 0 {
			cap.Seek(startFrame)
		}
	}

	if !cap.Read() {
		return Result{}, &trackError{"Erster Frame konnte nicht gelesen werden"}
	}

	tracker := NewTracker()
	// Schließt-was-"tracker"-GERADE-IST bei Rückkehr - kein "defer
	// tracker.Close()": das würde den POINTERWERT von tracker schon bei der
	// defer-Anweisung selbst festhalten, nicht bei ihrer Ausführung. Die
	// Schleife unten ersetzt tracker bei jedem Szenenschnitt (schließt den
	// alten explizit, verankert neu) - mit dem einfachen defer würde der
	// ALLERERSTE Tracker am Funktionsende ein zweites Mal geschlossen
	// (Doppel-Free, per SIGABRT/"double free or corruption" gefunden).
	defer func() { tracker.Close() }()
	tracker.Init(cap, roi)

	needsGray := opts.CameraCompensation || opts.SceneCutDetection || opts.AppearanceMemory || opts.RhythmGrid
	var prevGray *Gray
	if needsGray {
		prevGray = cap.ToGray()
		defer func() {
			if prevGray != nil {
				prevGray.Close()
			}
		}()
	}

	var memory *appearanceMemory
	if opts.AppearanceMemory {
		memory = newAppearanceMemory()
		defer memory.close()
		// Seed templates[0] with the ACTUAL frame-0 crop (prevGray still
		// holds it - needsGray is true whenever AppearanceMemory is, so
		// it's always available here). remember()'s own comment already
		// calls templates[0] "the user-confirmed start region"; without
		// this call that was only true once the tracker happened to
		// survive to the first periodic remember() ~1s in, already after
		// however much it had drifted by then.
		if prevGray != nil {
			memory.remember(prevGray, roi)
		}
	}

	timestampsMs := []int{0}
	yPositions := []float64{float64(roi.Y) + float64(roi.H)/2.0}
	xPositions := []float64{float64(roi.X) + float64(roi.W)/2.0}
	cameraDyCumulative := []float64{0.0}
	gridRows := rhythmGridRows(width, height)
	var flowX, flowY [][]float32 // per frame, only with opts.RhythmGrid
	if opts.RhythmGrid {
		flowX = [][]float32{make([]float32, rhythmGridCols*gridRows)}
		flowY = [][]float32{make([]float32, rhythmGridCols*gridRows)}
	}
	lastBbox := roi
	var guard dispGuard
	trackerLostFrames := 0
	cameraFramesLost := 0
	var sceneCuts []int

	totalFrames := int(cap.Get(CapPropFrameCount))
	if opts.MaxFrames > 0 && (totalFrames == 0 || opts.MaxFrames < totalFrames) {
		totalFrames = opts.MaxFrames
	}
	prog := newProgressReporter(opts.OnProgress, totalFrames)
	prog.report(0)

	frameIdx := 1
	validFrames := 1 // frame 0 was Init
	for cap.Read() {
		if opts.Cancel != nil && opts.Cancel() {
			return Result{Canceled: true}, ErrCanceled
		}
		var gray *Gray
		if needsGray {
			gray = cap.ToGray()
		}

		isCut := false
		if opts.SceneCutDetection && prevGray != nil {
			isCut = HistCorrelation(prevGray, gray) < sceneCutHistThreshold ||
				SignatureDiff(prevGray, gray) > sceneCutDiffThreshold
		}

		var ok bool
		var bbox Rect
		if isCut {
			sceneCuts = append(sceneCuts, frameIdx)
			anchor := lastBbox
			if memory != nil {
				if found, reacquired := memory.reacquire(gray); reacquired {
					anchor = found
				}
			}
			tracker.Close()
			tracker = NewTracker()
			tracker.Init(cap, anchor)
			ok, bbox = true, anchor
		} else {
			bbox, ok = tracker.Update(cap)
			if ok {
				if d := math.Hypot(float64(bbox.X+bbox.W/2-lastBbox.X-lastBbox.W/2),
					float64(bbox.Y+bbox.H/2-lastBbox.Y-lastBbox.H/2)); guard.implausible(d) {
					// CSRT reported success but the box moved implausibly
					// far for a single frame - see dispGuard's package
					// comment. Treat it exactly like a loss so the same
					// reacquire-or-coast path below handles it, instead of
					// silently accepting a jump onto the wrong target.
					ok = false
				} else {
					guard.accept(d)
				}
			}
			if !ok && memory != nil {
				if found, reacquired := memory.reacquire(gray); reacquired {
					tracker.Close()
					tracker = NewTracker()
					tracker.Init(cap, found)
					ok, bbox = true, found
				}
			}
		}

		if !ok {
			trackerLostFrames++
			yPositions = append(yPositions, yPositions[len(yPositions)-1])
			xPositions = append(xPositions, xPositions[len(xPositions)-1])
		} else {
			validFrames++
			yPositions = append(yPositions, float64(bbox.Y)+float64(bbox.H)/2.0)
			xPositions = append(xPositions, float64(bbox.X)+float64(bbox.W)/2.0)
			lastBbox = bbox
			if memory != nil && frameIdx%rememberEveryNFrames == 0 {
				// Only add this crop to the memory bank if it still
				// resembles where tracking started - otherwise a slow
				// drift (see matchesOriginal's comment) keeps feeding the
				// bank crops of whatever the tracker has wandered onto,
				// so a later genuine loss reacquires onto that same wrong
				// spot instead of recovering the real target.
				if score, known := memory.matchesOriginal(gray, bbox); !known || score >= memory.minScore {
					memory.remember(gray, bbox)
				}
			}
		}

		if opts.CameraCompensation {
			if isCut {
				cameraDyCumulative = append(cameraDyCumulative, 0.0)
			} else {
				dy := EstimateCameraMotionY(prevGray, gray, lastBbox)
				if dy == 0.0 {
					cameraFramesLost++
				}
				cameraDyCumulative = append(cameraDyCumulative, cameraDyCumulative[len(cameraDyCumulative)-1]+dy)
			}
		} else {
			cameraDyCumulative = append(cameraDyCumulative, 0.0)
		}

		if opts.RhythmGrid {
			if isCut {
				// Motion across a cut is not motion.
				flowX = append(flowX, make([]float32, rhythmGridCols*gridRows))
				flowY = append(flowY, make([]float32, rhythmGridCols*gridRows))
			} else {
				vx, vy := FlowCells(prevGray, gray, rhythmGridCols, gridRows)
				flowX = append(flowX, vx)
				flowY = append(flowY, vy)
			}
		}

		if needsGray {
			prevGray.Close()
			prevGray = gray
		}

		timestampsMs = append(timestampsMs, int(float64(frameIdx)*1000.0/fps))
		prog.report(frameIdx)
		frameIdx++
		if opts.MaxFrames > 0 && frameIdx >= opts.MaxFrames {
			break
		}
	}
	prog.report(frameIdx)

	if opts.CameraCompensation {
		applyCameraCompensation(yPositions, cameraDyCumulative, sceneCuts)
	}

	if opts.StartTimeSec > 0 {
		off := int(opts.StartTimeSec*1000 + 0.5)
		for i := range timestampsMs {
			timestampsMs[i] += off
		}
	}

	verticalRange := ptp(yPositions)
	horizontalRange := ptp(xPositions)
	axisIsHorizontal := horizontalRange > verticalRange*1.5 && horizontalRange > 5

	useX := opts.Axis == "x" || (opts.Axis != "y" && axisIsHorizontal)
	positions := yPositions
	if useX {
		positions = xPositions
	}
	if opts.RhythmGrid {
		cellV := flowY
		if useX {
			cellV = flowX
		}
		positions = rhythmGridPositions(cellV, rhythmGridCols, gridRows, width, height,
			xPositions, yPositions, positions, fps)
	}

	confidence := 0.0
	if frameIdx > 0 {
		confidence = float64(validFrames) / float64(frameIdx)
	}
	reason := ""
	lostFrac := 0.0
	if frameIdx > 0 {
		lostFrac = float64(trackerLostFrames) / float64(frameIdx)
	}
	if lostFrac > 0.5 {
		reason = "tracker_lost_heavy"
	} else if lostFrac > 0.15 {
		reason = "tracker_lost_elevated"
	}

	var trajA []Point
	if opts.CaptureTrajectory {
		// xPositions/yPositions already track the box center each frame
		// (camera-compensated on Y when enabled) - just zip them.
		trajA = make([]Point, len(xPositions))
		for i := range xPositions {
			trajA[i] = Point{X: xPositions[i], Y: yPositions[i]}
		}
	}

	return Result{
		TimestampsMs: timestampsMs,
		Positions:    positions,
		Width:        width,
		Height:       height,
		SceneCuts:    sceneCuts,
		TrajectoryA:  trajA,
		Stats: Stats{
			TrackerLostFrames: trackerLostFrames,
			CameraFramesLost:  cameraFramesLost,
			TotalFrames:       frameIdx,
			VerticalRange:     verticalRange,
			HorizontalRange:   horizontalRange,
			ValidFrames:       validFrames,
			Confidence:        confidence,
			Reason:            reason,
		},
	}, nil
}

// applyCameraCompensation glättet camera_dy SEGMENTWEISE (zwischen
// Szenenschnitten) und zieht es von yPositions ab - in place, wie das
// Python-Original. Segmentweise, nicht über die ganze Kurve auf einmal:
// sonst würde die Glättung selbst wieder Drift über eine Schnittgrenze
// hinweg mischen, die gerade bewusst auf 0 zurückgesetzt wurde.
func applyCameraCompensation(yPositions, cameraDy []float64, sceneCuts []int) {
	bounds := append([]int{0}, sceneCuts...)
	bounds = append(bounds, len(cameraDy))
	for i := 0; i < len(bounds)-1; i++ {
		start, end := bounds[i], bounds[i+1]
		if end-start >= 9 {
			smoothed := savgol(cameraDy[start:end], 9, 2)
			copy(cameraDy[start:end], smoothed)
		}
	}
	for i := range yPositions {
		yPositions[i] -= cameraDy[i]
	}
}

func ptp(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	lo, hi := v[0], v[0]
	for _, x := range v {
		lo = math.Min(lo, x)
		hi = math.Max(hi, x)
	}
	return hi - lo
}

type trackError struct{ msg string }

func (e *trackError) Error() string { return e.msg }
