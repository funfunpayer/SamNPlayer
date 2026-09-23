package generator

import "github.com/funfunpayer/SamNPlayer/funscript"

// buildContactMarksMeta returns nil when there is nothing to store.
func buildContactMarksMeta(opts Options) *funscript.ContactMarks {
	tip := opts.RegionClass
	var primary *funscript.ContactMarkBox
	if opts.ROI2.W > 0 && opts.ROI2.H > 0 {
		primary = &funscript.ContactMarkBox{
			X: opts.ROI2.X, Y: opts.ROI2.Y, W: opts.ROI2.W, H: opts.ROI2.H,
			Class: opts.RegionClass2,
			Fixed: opts.ROI2Fixed,
		}
	}
	var extras []funscript.ContactMarkBox
	for _, t := range opts.ExtraTargets {
		if t.W <= 0 || t.H <= 0 {
			continue
		}
		extras = append(extras, funscript.ContactMarkBox{
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
	return &funscript.ContactMarks{
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
