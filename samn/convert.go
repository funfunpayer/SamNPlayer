package samn

import (
	"encoding/json"
	"os"

	"github.com/funfunpayer/SamNPlayer/funscript"
)

// FromFunscript builds a Document from a community/legacy .funscript.
// Does not invent vibration/suction curves — playbackSource stays recipe
// unless the funscript already carried samn_axes + playback_source=axes.
func FromFunscript(s *funscript.Script, videoPath string) *Document {
	d := &Document{
		Version:    CurrentVersion,
		Kind:       Kind,
		VideoPath:  videoPath,
		Creator:    s.Metadata.Creator,
		DurationMs: s.Metadata.Duration,
		Profile:    s.Metadata.Profile,
		General:    append([]Point(nil), s.Actions...),
	}
	if d.DurationMs <= 0 {
		d.DurationMs = s.Duration()
	}
	if dr := s.Metadata.DeviceRecipe; dr != nil {
		d.Recipe = *dr
		d.PlaybackSource = funscript.NormalizePlaybackSource(dr.PlaybackSource)
	} else {
		d.PlaybackSource = PlaybackRecipe
	}
	if ax := s.Metadata.SamnAxes; ax != nil {
		d.Vibration = append([]Point(nil), ax.Vibration...)
		d.Suction = append([]Point(nil), ax.Suction...)
	}
	d.TrackingGaps = append([]funscript.TrackingGap(nil), s.Metadata.TrackingGaps...)
	d.QualityScore = s.Metadata.QualityScore
	d.QualityPassed = s.Metadata.QualityPassed
	d.QualityWarnings = append([]string(nil), s.Metadata.QualityWarnings...)
	return d
}

// LoadFunscriptFile loads a .funscript and converts to Document.
func LoadFunscriptFile(path string) (*Document, error) {
	s, err := funscript.Load(path)
	if err != nil {
		return nil, err
	}
	d := FromFunscript(s, "")
	// Chapters/bookmarks live outside the typed Script metadata — load separately.
	if ch, err := funscript.LoadChapters(path); err == nil {
		d.Chapters = ch
	}
	if bm, err := funscript.LoadBookmarks(path); err == nil {
		d.Bookmarks = bm
	}
	if om, err := funscript.LoadOMarkers(path); err == nil {
		d.OMarkers = om
	}
	_ = d.Normalize()
	return d, nil
}

// ToFunscript builds an in-memory funscript.Script for the player/editor
// bridge (includes samn_axes when present).
func (d *Document) ToFunscript() (*funscript.Script, error) {
	if err := d.Normalize(); err != nil {
		return nil, err
	}
	s := &funscript.Script{Actions: append([]funscript.Action(nil), d.General...)}
	s.Metadata.Creator = d.Creator
	s.Metadata.Duration = d.DurationMs
	s.Metadata.Profile = d.Profile
	s.Metadata.QualityScore = d.QualityScore
	s.Metadata.QualityPassed = d.QualityPassed
	s.Metadata.QualityWarnings = append([]string(nil), d.QualityWarnings...)
	s.Metadata.TrackingGaps = append([]funscript.TrackingGap(nil), d.TrackingGaps...)

	recipe := d.Recipe
	recipe.PlaybackSource = funscript.NormalizePlaybackSource(d.PlaybackSource)
	if recipe.Sync == "" && funscript.IsDistanceProfile(d.Profile) {
		meta := funscript.RecipeMeta(d.Profile)
		if recipe.TickMs == 0 {
			recipe = meta
		} else {
			recipe.Sync = meta.Sync
		}
		recipe.PlaybackSource = funscript.NormalizePlaybackSource(d.PlaybackSource)
	}
	s.Metadata.DeviceRecipe = &recipe

	if len(d.Vibration) > 0 || len(d.Suction) > 0 {
		s.Metadata.SamnAxes = &funscript.SamnAxes{
			Vibration: append([]funscript.Action(nil), d.Vibration...),
			Suction:   append([]funscript.Action(nil), d.Suction...),
		}
	}
	return s, nil
}

// ExportFunscript writes a community .funscript: general → actions,
// chapters/bookmarks in metadata. Neo-2 axes are NOT written (by design).
// Recipe is included so SamNPlayer can re-import contact settings.
func (d *Document) ExportFunscript(path string) error {
	if err := d.Normalize(); err != nil {
		return err
	}
	meta := map[string]any{
		"creator":  d.Creator,
		"duration": d.DurationMs,
	}
	if d.Profile != "" {
		meta["profile"] = d.Profile
	}
	if d.Recipe.Sync != "" || d.Recipe.ContactVibration || d.Recipe.TickMs > 0 {
		rec := d.Recipe
		// Export always uses recipe playback for foreign tools; drop axes source.
		rec.PlaybackSource = ""
		meta["device_recipe"] = rec
	}
	if len(d.TrackingGaps) > 0 {
		meta["tracking_gaps"] = d.TrackingGaps
	}
	if d.QualityScore != nil {
		meta["quality_score"] = *d.QualityScore
	}
	if d.QualityPassed != nil {
		meta["quality_passed"] = *d.QualityPassed
	}
	if len(d.QualityWarnings) > 0 {
		meta["quality_warnings"] = d.QualityWarnings
	}
	if len(d.Chapters) > 0 {
		meta["chapters"] = d.Chapters
	}
	if len(d.Bookmarks) > 0 {
		meta["bookmarks"] = d.Bookmarks
	}
	if len(d.OMarkers) > 0 {
		meta["oMarkers"] = d.OMarkers
	}
	doc := map[string]any{
		"actions":  d.General,
		"metadata": meta,
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// BakeNeoAxes fills vibration/suction from the recipe mapper and sets
// playbackSource to axes. Used after Tf/Tj (+ contact) generation.
func (d *Document) BakeNeoAxes() error {
	if err := d.Normalize(); err != nil {
		return err
	}
	opts := funscript.RecipeFor(d.Profile)
	if d.Recipe.Sync != "" {
		if sync, err := funscript.ParseSyncMode(d.Recipe.Sync); err == nil {
			opts.Sync = sync
		}
	}
	if d.Recipe.TickMs > 0 {
		opts.TickMs = d.Recipe.TickMs
	}
	if d.Recipe.MaxSpeed > 0 {
		opts.MaxSpeed = d.Recipe.MaxSpeed
	}
	opts.Smoothing = 0 // bake without recipe smoothing blur
	opts.MinSuction = d.Recipe.MinSuction
	opts.ContactVibration = d.Recipe.ContactVibration
	opts.ContactVibrationSpan = d.Recipe.ContactVibrationSpan
	opts.ContactVibrationCurve = d.Recipe.ContactVibrationCurve
	opts.TrackingGaps = d.TrackingGaps
	opts.ContactVibrationEnvelope = -1

	axes := funscript.BakeAxesFromRecipe(d.General, opts, 100)
	d.Vibration = axes.Vibration
	d.Suction = axes.Suction
	d.PlaybackSource = PlaybackAxes
	d.Recipe.PlaybackSource = PlaybackAxes
	return nil
}

// ActivePreset returns the selected strength preset, or nil.
func (d *Document) ActivePreset() *StrengthPreset {
	if d == nil || d.ActiveStrength == "" {
		return nil
	}
	for i := range d.StrengthPresets {
		if d.StrengthPresets[i].Name == d.ActiveStrength {
			return &d.StrengthPresets[i]
		}
	}
	return nil
}

// ApplyContactRecipe updates Tf/Tj contact-vibration fields on the recipe.
// When playbackSource is axes, call BakeNeoAxes afterwards so the vibration
// curve matches the new contact settings.
func (d *Document) ApplyContactRecipe(enabled bool, span float64, curve string) {
	if d.Recipe.Sync == "" {
		d.Recipe.Sync = funscript.SyncSuctionPosition.String()
	}
	d.Recipe.ContactVibration = enabled
	if enabled {
		d.Recipe.ContactVibrationSpan = funscript.EffectiveContactSpan(span)
		d.Recipe.ContactVibrationCurve = funscript.NormalizeContactCurve(curve)
	} else {
		d.Recipe.ContactVibrationSpan = 0
		d.Recipe.ContactVibrationCurve = ""
	}
}

// DefaultStrengthPresets soft/normal/strong for new Tf/Tj documents.
func DefaultStrengthPresets() []StrengthPreset {
	return []StrengthPreset{
		{Name: "soft", VibrationScale: 0.7, SuctionScale: 0.85, ContactSpan: 0.85, ContactCurve: "soft"},
		{Name: "normal", VibrationScale: 1.0, SuctionScale: 1.0},
		{Name: "strong", VibrationScale: 1.3, SuctionScale: 1.1, ContactSpan: 0.55, ContactCurve: "peak"},
	}
}
