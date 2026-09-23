package generator

import (
	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/samn"
)

// writeCompanionSamn writes clip.samn beside a generated .funscript.
// Play prefers this companion — so contact recipe, contact_marks, and
// tip trajectory must live here too (not only on the .funscript).
func writeCompanionSamn(funscriptPath string, actions []funscript.Action, opts Options, gaps []funscript.TrackingGap, quality funscript.ScriptQualityResult, traj *funscript.TrajectoryData) error {
	doc := &samn.Document{
		Version:        samn.CurrentVersion,
		Kind:           samn.Kind,
		Creator:        "SamNPlayer",
		DurationMs:     0,
		Profile:        funscript.NormalizeProfile(opts.Profile),
		General:        append([]funscript.Action(nil), actions...),
		PlaybackSource: samn.PlaybackRecipe,
	}
	if len(actions) > 0 {
		doc.DurationMs = actions[len(actions)-1].At
	}
	if funscript.IsDistanceProfile(opts.Profile) {
		recipe := funscript.RecipeMeta(opts.Profile)
		applyContactVibrationToRecipe(&recipe, opts)
		doc.Recipe = recipe
		doc.TrackingGaps = gaps
		doc.StrengthPresets = samn.DefaultStrengthPresets()
		doc.ActiveStrength = "normal"
		if err := doc.BakeNeoAxes(); err != nil {
			return err
		}
	} else if opts.ContactVibration {
		// Everyday Stroke + Contact vib: same recipe native.go stamps on .funscript.
		recipe := funscript.DeviceRecipe{
			Sync:      funscript.SyncIndependent.String(),
			Smoothing: 0.3,
		}
		if funscript.NormalizeProfile(opts.Profile) == funscript.ProfileWeich {
			recipe = funscript.RecipeMeta(funscript.ProfileWeich)
		}
		applyContactVibrationToRecipe(&recipe, opts)
		doc.Recipe = recipe
		doc.Profile = funscript.NormalizeProfile(opts.Profile)
		if doc.Profile == "" {
			doc.Profile = "standard"
		}
		doc.TrackingGaps = gaps
	} else if len(gaps) > 0 {
		doc.TrackingGaps = gaps
	}
	if cm := buildContactMarksMeta(opts); cm != nil {
		doc.ContactMarks = cm
	}
	if traj != nil && len(traj.Tip) > 0 {
		cp := *traj
		cp.Tip = append([]funscript.TrajectoryPoint(nil), traj.Tip...)
		cp.Partner = append([]funscript.TrajectoryPoint(nil), traj.Partner...)
		doc.Trajectory = &cp
	}
	if quality.Score > 0 || len(quality.Warnings) > 0 {
		score := quality.Score
		passed := quality.Passed
		doc.QualityScore = &score
		doc.QualityPassed = &passed
		doc.QualityWarnings = append([]string(nil), quality.Warnings...)
	}
	return samn.Save(samn.CompanionSamnPath(funscriptPath), doc)
}

func applyContactVibrationToRecipe(recipe *funscript.DeviceRecipe, opts Options) {
	if recipe == nil || !opts.ContactVibration {
		return
	}
	recipe.ContactVibration = true
	if opts.ContactVibrationSpan > 0 {
		span := funscript.EffectiveContactSpan(opts.ContactVibrationSpan)
		if span != funscript.DefaultContactVibrationSpan {
			recipe.ContactVibrationSpan = span
		}
	}
	curve := funscript.NormalizeContactCurve(opts.ContactVibrationCurve)
	if curve != funscript.ContactCurveLinear {
		recipe.ContactVibrationCurve = curve
	}
}
