// Package aiscript is the opt-in AI draft script path (Owner 26 Sep 2026).
//
// Everyday Create still uses classical CSRT. This package only becomes
// Available when a local draft model is configured; until then Status
// explains why the GUI control stays disabled.
//
// See docs/AI_SCRIPT_WRITER.md.
package aiscript

import "fmt"

// Status reports whether a local AI draft writer can run.
type Status struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
	ModelPath string `json:"modelPath,omitempty"`
	Stage     string `json:"stage"` // e.g. "S0", "S2"
}

// DraftRequest is the input for an experimental AI stroke draft.
type DraftRequest struct {
	VideoPath string `json:"videoPath"`
	ModelPath string `json:"modelPath,omitempty"`
	// TipROI optional seed box in pixel coords (x,y,w,h). Zero = unused.
	TipX float64 `json:"tipX,omitempty"`
	TipY float64 `json:"tipY,omitempty"`
	TipW float64 `json:"tipW,omitempty"`
	TipH float64 `json:"tipH,omitempty"`
}

// DraftResult is a proposed stroke only — caller must run Quality Doctor
// and require an explicit Keep in the GUI before replacing a script.
type DraftResult struct {
	Actions  []Action `json:"actions"`
	Warnings []string `json:"warnings,omitempty"`
	Engine   string   `json:"engine"`
	Notes    string   `json:"notes,omitempty"`
}

// Action mirrors funscript time/pos without importing funscript here
// (keeps the package leaf-light for early stages).
type Action struct {
	At  int64 `json:"at"`
	Pos int   `json:"pos"`
}

// StatusFor returns availability. modelPath empty → not configured (S0).
func StatusFor(modelPath string) Status {
	if modelPath == "" {
		return Status{
			Available: false,
			Reason:    "No AI draft model yet. You can still export classical good runs for training (S1). Everyday Create stays CSRT.",
			Stage:     "S1",
		}
	}
	// S2 will check file exists + format. Until then refuse any path so we
	// never pretend an unfinished writer works.
	return Status{
		Available: false,
		Reason:    "AI draft model path set, but draft inference is not implemented yet (stage S1→S2). Everyday CSRT unchanged.",
		ModelPath: modelPath,
		Stage:     "S1",
	}
}

// Draft runs the experimental writer. S0 always fails closed.
func Draft(req DraftRequest) (DraftResult, error) {
	st := StatusFor(req.ModelPath)
	if !st.Available {
		return DraftResult{}, fmt.Errorf("aiscript: %s", st.Reason)
	}
	return DraftResult{}, fmt.Errorf("aiscript: unreachable")
}
