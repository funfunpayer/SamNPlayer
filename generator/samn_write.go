package generator

import (
	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/samn"
)

// writeCompanionSamn writes clip.samn beside a generated .funscript.
func writeCompanionSamn(funscriptPath string, actions []funscript.Action, opts Options, gaps []funscript.TrackingGap, quality funscript.ScriptQualityResult) error {
	doc := &samn.Document{
		Version:    samn.CurrentVersion,
		Kind:       samn.Kind,
		Creator:    "SamNPlayer",
		DurationMs: 0,
		Profile:    funscript.NormalizeProfile(opts.Profile),
		General:    append([]funscript.Action(nil), actions...),
		PlaybackSource: samn.PlaybackRecipe,
	}
	if len(actions) > 0 {
		doc.DurationMs = actions[len(actions)-1].At
	}
	if funscript.IsDistanceProfile(opts.Profile) {
		recipe := funscript.RecipeMeta(opts.Profile)
		if opts.ContactVibration {
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
		doc.Recipe = recipe
		doc.TrackingGaps = gaps
		doc.StrengthPresets = samn.DefaultStrengthPresets()
		doc.ActiveStrength = "normal"
		if err := doc.BakeNeoAxes(); err != nil {
			return err
		}
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
