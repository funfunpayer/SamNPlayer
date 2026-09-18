package funscript

// PipelineSuggestion is an automatic preselection for generate options.
type PipelineSuggestion struct {
	Backend string `json:"Backend"`
	Profile string `json:"Profile"`
	Reason  string `json:"Reason"`
	GoPath  bool   `json:"GoPath"`
}

// SuggestPipeline picks backend + profile from marked ROI geometry.
// Two valid ROIs → Tf/Tj + CSRT (Go two-point). Small single ROI → grid_lk
// (Python). Large single ROI → CSRT (Go). Never invents flow.
func SuggestPipeline(roiW, roiH, roi2W, roi2H int) PipelineSuggestion {
	if roi2W > 0 && roi2H > 0 && roiW > 0 && roiH > 0 {
		return PipelineSuggestion{
			Backend: "csrt",
			Profile: "tf",
			Reason:  "Zwei Regionen → Tf/Tj (Abstand) auf der Go-Pipeline",
			GoPath:  true,
		}
	}
	if roiW <= 0 || roiH <= 0 {
		return PipelineSuggestion{
			Backend: "csrt",
			Profile: "standard",
			Reason:  "Keine Region — CSRT-Standard (Region noch markieren)",
			GoPath:  false,
		}
	}
	backend := SuggestBackend(roiW, roiH)
	goPath := backend == "csrt"
	reason := "Eine Region + CSRT → Go-Pipeline"
	if backend == "grid_lk" {
		reason = "Kleine Region → Gitter/Optical-Flow (Python; gemessen robuster bei kleinen Boxen)"
	}
	return PipelineSuggestion{
		Backend: backend,
		Profile: "standard",
		Reason:  reason,
		GoPath:  goPath,
	}
}
