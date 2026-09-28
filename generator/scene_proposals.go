package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// SceneProposal is one window of `scene_roles.py` output (scene
// understanding stage 2): the part that moves most is the primary stroke
// target, the part it moves against is the contact partner, and the pair
// gives the scene type. These are proposals only - the user applies them
// (TFTJ rules: no silent ROI2).
type SceneProposal struct {
	TMs        int64         `json:"t_ms"`
	StartMs    int64         `json:"start_ms"`
	EndMs      int64         `json:"end_ms"`
	SceneType  string        `json:"scene_type"` // blowjob|handjob|titjob|penetration, "" = unknown
	Confidence float64       `json:"confidence"`
	Primary    ROICandidate  `json:"primary"`
	Partner    *ROICandidate `json:"partner,omitempty"`
}

// SceneProposals is a loaded `<clip>.scene.json`.
type SceneProposals struct {
	Width     int             `json:"width"`
	Height    int             `json:"height"`
	Proposals []SceneProposal `json:"proposals"`
}

// LoadSceneProposals reads `<clip>.scene.json` from scene_roles.py.
// Proposals are sorted by time; boxes outside the frame or empty are
// dropped with the proposal (primary) or on their own (partner).
func LoadSceneProposals(path string) (SceneProposals, error) {
	var s SceneProposals
	b, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	var raw struct {
		Version int `json:"version"`
		SceneProposals
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return s, fmt.Errorf("generator: scene proposals %s: %w", path, err)
	}
	if raw.Version != 1 {
		return s, fmt.Errorf("generator: scene proposals %s: unsupported version %d", path, raw.Version)
	}
	if raw.Width <= 0 || raw.Height <= 0 {
		return s, fmt.Errorf("generator: scene proposals %s: missing width/height", path)
	}
	s.Width, s.Height = raw.Width, raw.Height
	in := func(c ROICandidate) bool {
		return c.W > 0 && c.H > 0 && c.X >= 0 && c.Y >= 0 && c.X+c.W <= s.Width && c.Y+c.H <= s.Height
	}
	for _, p := range raw.Proposals {
		if !in(p.Primary) {
			continue
		}
		p.Primary.Index = 1
		if p.Partner != nil {
			if in(*p.Partner) {
				p.Partner.Index = 2
			} else {
				p.Partner = nil
			}
		}
		s.Proposals = append(s.Proposals, p)
	}
	sort.SliceStable(s.Proposals, func(i, j int) bool { return s.Proposals[i].TMs < s.Proposals[j].TMs })
	return s, nil
}

// At returns the proposal whose window covers atMs, else the nearest one by
// window centre; ok is false when there are none.
func (s SceneProposals) At(atMs int64) (SceneProposal, bool) {
	best, bestD := -1, int64(0)
	for i, p := range s.Proposals {
		if p.EndMs > p.StartMs && atMs >= p.StartMs && atMs < p.EndMs {
			return p, true
		}
		d := p.TMs - atMs
		if d < 0 {
			d = -d
		}
		if best < 0 || d < bestD {
			best, bestD = i, d
		}
	}
	if best < 0 {
		return SceneProposal{}, false
	}
	return s.Proposals[best], true
}
