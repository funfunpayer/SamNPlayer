package trackcv

// Rect is a pixel-space axis-aligned box (x, y, w, h).
type Rect struct{ X, Y, W, H int }

// SceneMark is a user/auto annotation that can filter rhythm-grid candidates
// (docs/SCENE_MAP_PLAN.md M3). Kind is "exclude", "source", or "region".
// FromMs/ToMs both 0 means the whole clip; otherwise active when
// FromMs <= t <= ToMs.
type SceneMark struct {
	Kind   string // exclude | source | region
	ID     string
	Rect   Rect
	FromMs int64
	ToMs   int64
	Class  string
}

// sceneMarkActive reports whether m applies at time midMs.
func sceneMarkActive(m SceneMark, midMs int64) bool {
	if m.FromMs == 0 && m.ToMs == 0 {
		return true
	}
	return midMs >= m.FromMs && midMs <= m.ToMs
}

func pointInRect(px, py float64, r Rect) bool {
	return px >= float64(r.X) && px < float64(r.X+r.W) &&
		py >= float64(r.Y) && py < float64(r.Y+r.H)
}

// cameraExcludeRects builds the feature punch-out list for camera motion:
// tracked box plus active exclude SceneMarks (incl. soft masks converted
// to exclude at the call site).
func cameraExcludeRects(tracked Rect, marks []SceneMark, atMs int64) []Rect {
	out := make([]Rect, 0, 1+len(marks))
	out = append(out, tracked)
	for _, mk := range marks {
		if mk.Kind != "exclude" || !sceneMarkActive(mk, atMs) {
			continue
		}
		out = append(out, mk.Rect)
	}
	return out
}
