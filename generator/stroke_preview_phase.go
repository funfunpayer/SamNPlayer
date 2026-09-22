package generator

import (
	"context"
	"fmt"

	"github.com/funfunpayer/SamNPlayer/generator/strokepreview"
	"github.com/funfunpayer/SamNPlayer/logging"
)

// applyStrokePreview runs Stage A before tracking and folds safe hints into
// opts (audio gate, optional peak-distance bias). Failures are non-fatal —
// Generate continues; we only lose the hint.
func applyStrokePreview(ctx context.Context, videoPath string, opts Options, onProgress func(string)) Options {
	if opts.SkipStrokePreview {
		return opts
	}
	if onProgress != nil {
		onProgress("STROKE_PREVIEW starting (sparse extrema / cut / pan)…")
	}
	rep, err := strokepreview.RunQuick(ctx, videoPath)
	if err != nil {
		msg := fmt.Sprintf("STROKE_PREVIEW skipped: %v", err)
		logging.Warn("generator: stroke preview failed", "error", err)
		if onProgress != nil {
			onProgress(msg)
		}
		return opts
	}
	h := rep.ToHint()
	if onProgress != nil {
		onProgress(h.ProgressLine())
	}
	opts.StrokePreviewHint = h.MetadataMap()

	// Audio only when preview looks bad — never invents positions.
	if h.SuggestAudioCheck && !opts.AudioCheck {
		opts.AudioCheck = true
		if onProgress != nil {
			onProgress("STROKE_PREVIEW: enabling audio tempo check (weak/unstable preview)")
		}
	}
	// Bias peak distance only when caller left it at profile default (0).
	if opts.MinPeakDistanceMs == 0 && h.SuggestedMinPeakDistanceMs > 0 &&
		(h.Quality == "ok" || h.Quality == "unstable") {
		opts.MinPeakDistanceMs = h.SuggestedMinPeakDistanceMs
		if onProgress != nil {
			onProgress(fmt.Sprintf("STROKE_PREVIEW: min peak distance → %dms (from ~%.2f Hz)",
				opts.MinPeakDistanceMs, h.StrokeHz))
		}
	}
	if h.CutRatePerMin > 4 && onProgress != nil {
		onProgress(fmt.Sprintf("STROKE_PREVIEW: high cut rate (%.1f/min) — consider “Re-find region after each cut”",
			h.CutRatePerMin))
	}
	if h.PanShare > 0.4 && onProgress != nil {
		onProgress(fmt.Sprintf("STROKE_PREVIEW: pan-like energy share=%.0f%% — camera compensation matters",
			h.PanShare*100))
	}
	return opts
}
