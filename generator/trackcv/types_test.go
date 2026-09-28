package trackcv

import "testing"

func TestSceneMarkRectAt_static(t *testing.T) {
	m := SceneMark{Rect: Rect{10, 20, 30, 40}}
	got := sceneMarkRectAt(m, 5000)
	if got != m.Rect {
		t.Fatalf("static rect: got %+v want %+v", got, m.Rect)
	}
}

func TestSceneMarkRectAt_nearestPath(t *testing.T) {
	m := SceneMark{
		Rect: Rect{0, 0, 10, 10},
		Path: []MarkSample{
			{Ms: 0, Rect: Rect{0, 0, 10, 10}},
			{Ms: 1000, Rect: Rect{50, 60, 10, 10}},
			{Ms: 2000, Rect: Rect{100, 120, 12, 12}},
		},
	}
	got := sceneMarkRectAt(m, 1100)
	want := Rect{50, 60, 10, 10}
	if got != want {
		t.Fatalf("nearest@1100: got %+v want %+v", got, want)
	}
	got = sceneMarkRectAt(m, 1900)
	want = Rect{100, 120, 12, 12}
	if got != want {
		t.Fatalf("nearest@1900: got %+v want %+v", got, want)
	}
}

func TestCameraExcludeRects_usesPath(t *testing.T) {
	tracked := Rect{200, 200, 40, 40}
	marks := []SceneMark{{
		Kind: "exclude", ID: "knee",
		Rect: Rect{0, 0, 20, 20},
		Path: []MarkSample{
			{Ms: 0, Rect: Rect{0, 0, 20, 20}},
			{Ms: 8000, Rect: Rect{80, 90, 20, 20}},
		},
	}}
	got := cameraExcludeRects(tracked, marks, 8000)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[1] != (Rect{80, 90, 20, 20}) {
		t.Fatalf("exclude path rect: %+v", got[1])
	}
}

func TestMarkShouldFollow(t *testing.T) {
	if markShouldFollow(SceneMark{Kind: "exclude", Follow: true, Rect: Rect{W: 10, H: 10}}) != true {
		t.Fatal("exclude+follow should track")
	}
	if markShouldFollow(SceneMark{Kind: "exclude", Follow: false, Rect: Rect{W: 10, H: 10}}) {
		t.Fatal("sticky exclude must not track")
	}
}

func TestSceneMarkTimebaseIncludesStartOffset(t *testing.T) {
	startTimeSec := 10.0
	mark := SceneMark{Kind: "exclude", FromMs: 10500, ToMs: 11500}

	atMs := int64(750)
	if startTimeSec > 0 {
		atMs += int64(startTimeSec*1000 + 0.5)
	}
	if !sceneMarkActive(mark, atMs) {
		t.Fatalf("mark should be active at absolute video time %dms", atMs)
	}

	relativeOnly := int64(750)
	if sceneMarkActive(mark, relativeOnly) {
		t.Fatalf("regression guard invalid: mark unexpectedly active at relative time %dms", relativeOnly)
	}
}
