package funscript

import (
	"sort"
)

// ChapterFeelSuggestion is a suggested scene feel for one chapter.
// Suggestions only — the user confirms before anything is written.
type ChapterFeelSuggestion struct {
	StartMs  int64  `json:"startMs"`
	EndMs    int64  `json:"endMs"`
	Name     string `json:"name,omitempty"`
	Feel     string `json:"feel"`     // oral | intercourse | breast | hand | ""
	Source   string `json:"source"`   // ai | motion | contact
	Reason   string `json:"reason,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

// SuggestChapterFeels builds scene-feel suggestions for each chapter.
// AI proposals (scene type at the chapter midpoint / majority) take priority.
// Motion / contact are classic fallbacks. Empty chapters or no signal → Classic (feel "").
func SuggestChapterFeels(chapters []ChapterMark, proposalsAt func(ms int64) (sceneType string, conf float64, ok bool), contactClass string) []ChapterFeelSuggestion {
	if len(chapters) == 0 {
		return nil
	}
	out := make([]ChapterFeelSuggestion, 0, len(chapters))
	for _, c := range chapters {
		start, end := c.StartTime, c.EndTime
		if end <= start {
			end = start + 1
		}
		mid := start + (end-start)/2
		sug := ChapterFeelSuggestion{
			StartMs: start,
			EndMs:   end,
			Name:    c.Name,
		}
		// AI first
		if proposalsAt != nil {
			if st, conf, ok := proposalsAt(mid); ok && st != "" {
				if feel := NormalizeSceneFeel(st); feel != "" {
					sug.Feel = feel
					sug.Source = "ai"
					sug.Confidence = conf
					sug.Reason = "scene detection: " + st
					out = append(out, sug)
					continue
				}
			}
		}
		// Classic: contact mark class
		if contactClass != "" {
			if feel := sceneFromContact(contactClass); feel != "" {
				sug.Feel = feel
				sug.Source = "contact"
				sug.Reason = "contact mark: " + contactClass
				out = append(out, sug)
				continue
			}
		}
		// Motion fallback (coarse): nothing strong → Classic
		sug.Source = "motion"
		sug.Reason = "no strong scene signal"
		out = append(out, sug)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartMs < out[j].StartMs })
	return out
}

func sceneFromContact(class string) string {
	switch NormalizeSceneFeel(class) {
	case SceneFeelOral, SceneFeelBreast, SceneFeelHand:
		return NormalizeSceneFeel(class)
	}
	return ""
}
