package funscript

import "fmt"

// OZoneSuggestion is a candidate range for a primary O-marker / Extended-O
// band. It is a proposal: the user confirms, moves, or discards it.
type OZoneSuggestion struct {
	StartMs int64  `json:"startMs"`
	EndMs   int64  `json:"endMs"`
	Reason  string `json:"reason"`
	OK      bool   `json:"ok"`
}

// SuggestOZone picks a window in the last 8–15% of the script that has the
// highest mean position among sliding windows of ~8% duration. That matches
// "Ende + anhaltend hohe Intensität" without claiming semantic recognition.
func SuggestOZone(actions []Action) OZoneSuggestion {
	if len(actions) < 4 {
		return OZoneSuggestion{Reason: "zu wenige Punkte"}
	}
	start, end := actions[0].At, actions[len(actions)-1].At
	dur := end - start
	if dur < 4000 {
		return OZoneSuggestion{Reason: "Skript kürzer als 4s — kein O-Vorschlag"}
	}
	tailStart := start + int64(float64(dur)*0.85)
	win := int64(float64(dur) * 0.08)
	if win < 1500 {
		win = 1500
	}
	if tailStart+win > end {
		tailStart = end - win
	}
	bestStart := tailStart
	bestMean := -1.0
	step := win / 4
	if step < 200 {
		step = 200
	}
	for w0 := tailStart; w0+win <= end; w0 += step {
		mean := windowMeanPos(actions, w0, w0+win)
		if mean > bestMean {
			bestMean = mean
			bestStart = w0
		}
	}
	if bestMean < 35 {
		return OZoneSuggestion{
			Reason: fmt.Sprintf("letztes Achtel bleibt flach (Mittel %.0f) — kein O-Vorschlag", bestMean),
		}
	}
	return OZoneSuggestion{
		StartMs: bestStart,
		EndMs:   bestStart + win,
		OK:      true,
		Reason: fmt.Sprintf(
			"letztes Achtel, Fenster mit höchster mittlerer Position (%.0f)",
			bestMean),
	}
}

func windowMeanPos(actions []Action, from, to int64) float64 {
	var sum float64
	var n int
	for _, a := range actions {
		if a.At >= from && a.At <= to {
			sum += float64(a.Pos)
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
