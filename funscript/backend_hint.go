package funscript

// SuggestBackend picks a tracker name from ROI size only.
// Small/blurry boxes measured better on grid_lk; large textured hubs on CSRT.
// Flow is never chosen here — it ignores the box.
func SuggestBackend(w, h int) string {
	if w <= 0 || h <= 0 {
		return "csrt"
	}
	area := w * h
	if area < 800 || w < 24 || h < 24 {
		return "grid_lk"
	}
	return "csrt"
}
