package generator

import "github.com/funfunpayer/SamNPlayer/funscript"

// ContactMarkBox is one optional contact area stored for later feel
// (nipples / mouth / …). Does not drive the Everyday stroke curve.
type ContactMarkBox struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	W     int    `json:"w"`
	H     int    `json:"h"`
	Class string `json:"class,omitempty"`
	Fixed bool   `json:"fixed,omitempty"`
}

// ContactMarksMeta is stamped into funscript metadata.contact_marks when the
// user marked tip/contact classes or areas with Contact vibration on.
// drive_stroke is false on Everyday CSRT — tip alone writes the curve;
// marks are for feel / future feel-decouple.
type ContactMarksMeta struct {
	TipClass    string           `json:"tip_class,omitempty"`
	Primary     *ContactMarkBox  `json:"primary,omitempty"`
	Extras      []ContactMarkBox `json:"extras,omitempty"`
	DriveStroke bool             `json:"drive_stroke"`
}

// buildContactMarksMeta returns nil when there is nothing to store.
func buildContactMarksMeta(opts Options) *ContactMarksMeta {
	tip := opts.RegionClass
	var primary *ContactMarkBox
	if opts.ROI2.W > 0 && opts.ROI2.H > 0 {
		primary = &ContactMarkBox{
			X: opts.ROI2.X, Y: opts.ROI2.Y, W: opts.ROI2.W, H: opts.ROI2.H,
			Class: opts.RegionClass2,
			Fixed: opts.ROI2Fixed,
		}
	}
	var extras []ContactMarkBox
	for _, t := range opts.ExtraTargets {
		if t.W <= 0 || t.H <= 0 {
			continue
		}
		extras = append(extras, ContactMarkBox{
			X: t.X, Y: t.Y, W: t.W, H: t.H,
			Class: t.Class,
			Fixed: t.Fixed,
		})
	}
	if tip == "" && primary == nil && len(extras) == 0 {
		return nil
	}
	// Distance profiles already use ROI2 for tip↔partner; still stamp marks
	// so classes / extras survive. Everyday stroke never drives from marks.
	drive := funscript.IsDistanceProfile(opts.Profile)
	return &ContactMarksMeta{
		TipClass:    tip,
		Primary:     primary,
		Extras:      extras,
		DriveStroke: drive,
	}
}

// stampContactMarks writes metadata.contact_marks when present.
func stampContactMarks(meta map[string]any, opts Options) {
	if meta == nil {
		return
	}
	cm := buildContactMarksMeta(opts)
	if cm == nil {
		return
	}
	meta["contact_marks"] = cm
}
