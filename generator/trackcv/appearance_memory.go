//go:build cgo && opencv && !windows

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
