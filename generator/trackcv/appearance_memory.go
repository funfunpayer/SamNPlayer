//go:build cgo && opencv

package trackcv

// appearanceMemory ist die Go-Entsprechung von generate_funscript.py's
// AppearanceMemory: merkt sich in Abständen kleine Graustufen-Ausschnitte
// der verfolgten Region, und findet sie per Template-Matching im ganzen
// Bild wieder, wenn der Tracker das Ziel verliert oder ein Szenenschnitt
// kommt - siehe dortiger Klassenkommentar für die Begründung (ein
// Objekterkenner-Ersatz ohne Modell).
type appearanceMemory struct {
	maxTemplates int
	minScore     float64
	downscale    float64
	templates    []*Gray
}

func newAppearanceMemory() *appearanceMemory {
	return &appearanceMemory{maxTemplates: 8, minScore: 0.55, downscale: 0.5}
}

func (m *appearanceMemory) close() {
	for _, t := range m.templates {
		t.Close()
	}
	m.templates = nil
}

func (m *appearanceMemory) remember(gray *Gray, box Rect) {
	if box.W <= 0 || box.H <= 0 {
		return
	}
	tmpl := gray.ExtractTemplate(box, m.downscale)
	if tmpl == nil {
		return
	}
	m.templates = append(m.templates, tmpl)
	if len(m.templates) > m.maxTemplates {
		// Das ÄLTESTE verwerfen, aber den ERSTEN (Index 0, die vom Nutzer
		// bestätigte Startregion) behalten - siehe Python-Original.
		m.templates[1].Close()
		m.templates = append(m.templates[:1], m.templates[2:]...)
	}
}

// matchesOriginal reports how well box (the tracker's CURRENT belief, in
// full-resolution gray coordinates) still resembles templates[0] - the
// very first remembered crop, kept forever by remember() specifically as
// the "user-confirmed start region" (see its comment above).
//
// Why this exists: TrackROI's CSRT can drift gradually off the true
// target over a long continuous run (a handful/no single frame ever
// moves far enough to look wrong, but the box still ends up somewhere
// else entirely after a couple of minutes - confirmed on a real clip,
// docs/AGENT_COORD.md 23 Sep "CSRT long-clip drift"). remember() runs
// periodically on whatever the tracker CURRENTLY believes, with no check
// against reality, so once drift sets in, the memory bank keeps filling
// with crops of the wrong region. When the tracker then genuinely loses
// the target (which it eventually did on that same clip), reacquire()
// searches for a match to those poisoned templates and confidently
// re-anchors on the SAME wrong region - not a bug in reacquire() itself,
// it faithfully found what it was told to look for.
//
// ok=false means "nothing to compare against yet" (no templates
// remembered) - callers should treat that as "allow it", not "reject
// it": there's no basis for suspicion this early.
func (m *appearanceMemory) matchesOriginal(gray *Gray, box Rect) (score float64, ok bool) {
	if len(m.templates) == 0 {
		return 0, false
	}
	orig := m.templates[0]
	// A local neighborhood around the current box, not another
	// full-frame search (that's reacquire()'s job) - large enough that
	// orig fits strictly inside per MatchTemplate's own requirement.
	region := gray.ExtractTemplate(Rect{
		X: box.X - box.W, Y: box.Y - box.H,
		W: box.W * 3, H: box.H * 3,
	}, m.downscale)
	if region == nil {
		return 0, false
	}
	defer region.Close()
	if orig.Width() >= region.Width() || orig.Height() >= region.Height() {
		return 0, false
	}
	_, _, matchScore, matched := MatchTemplate(region, orig)
	return matchScore, matched
}

// reacquire sucht die Region im ganzen Bild. ok=false, wenn nichts
// Verlässliches gefunden wurde (unterhalb minScore wird NICHT neu
// verankert - eine geratene Position ist schlechter als die alte).
func (m *appearanceMemory) reacquire(gray *Gray) (box Rect, ok bool) {
	if len(m.templates) == 0 {
		return Rect{}, false
	}
	frame := gray.Downscale(m.downscale)
	defer frame.Close()

	bestScore := -1.0
	var bestX, bestY, bestW, bestH int
	found := false
	for _, tmpl := range m.templates {
		if tmpl.Height() >= frame.Height() || tmpl.Width() >= frame.Width() {
			continue
		}
		x, y, score, matched := MatchTemplate(frame, tmpl)
		if !matched {
			continue
		}
		if score > bestScore {
			bestScore, bestX, bestY = score, x, y
			bestW, bestH = tmpl.Width(), tmpl.Height()
			found = true
		}
	}

	if !found || bestScore < m.minScore {
		return Rect{}, false
	}

	scale := 1.0
	if m.downscale != 0 {
		scale = 1.0 / m.downscale
	}
	x := int(float64(bestX) * scale)
	y := int(float64(bestY) * scale)
	w := maxInt(8, int(float64(bestW)*scale))
	h := maxInt(8, int(float64(bestH)*scale))
	return Rect{X: x, Y: y, W: w, H: h}, true
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
