package funscript

// PipelineSuggestion is an automatic preselection for generate options.
type PipelineSuggestion struct {
	Backend string `json:"Backend"`
	Profile string `json:"Profile"`
	Reason  string `json:"Reason"`
	GoPath  bool   `json:"GoPath"`
}

// SuggestPipeline picks backend + profile from marked ROI geometry.
// Always CSRT so the default path stays Go (no Python). Two valid ROIs →
// Tf/Tj. Research backends remain CLI --backend only.
func SuggestPipeline(roiW, roiH, roi2W, roi2H int) PipelineSuggestion {
	if roi2W > 0 && roi2H > 0 && roiW > 0 && roiH > 0 {
		return PipelineSuggestion{
			Backend: "csrt",
			Profile: "tf",
			Reason:  "Two regions → Tf/Tj (distance) on the Go path",
			GoPath:  true,
		}
	}
	if roiW <= 0 || roiH <= 0 {
		return PipelineSuggestion{
			Backend: "csrt",
			Profile: "standard",
			Reason:  "No region yet — mark a region, then CSRT (Go path)",
			GoPath:  true,
		}
	}
	_ = SuggestBackend(roiW, roiH)
	return PipelineSuggestion{
		Backend: "csrt",
		Profile: "standard",
		Reason:  "One region + CSRT → Go path (no Python)",
		GoPath:  true,
	}
}
