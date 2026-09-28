package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoiFromSceneProposalsLogsAndNeverSetsROI2(t *testing.T) {
	p := filepath.Join(t.TempDir(), "clip.scene.json")
	body := `{"version":1,"width":1280,"height":720,"proposals":[
	 {"t_ms":4000,"start_ms":0,"end_ms":8000,"scene_type":"blowjob","confidence":0.8,
	  "primary":{"x":560,"y":400,"w":160,"h":80,"score":0.8,"class":"mouth"},
	  "partner":{"x":560,"y":480,"w":80,"h":240,"score":0.8,"class":"penis"}},
	 {"t_ms":14000,"start_ms":10000,"end_ms":18000,"scene_type":"penetration","confidence":0.6,
	  "primary":{"x":100,"y":100,"w":80,"h":80,"score":0.6,"class":"buttocks"}}]}`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var log []string
	roi, class, err := roiFromSceneProposals(p, 0, func(l string) { log = append(log, l) })
	if err != nil {
		t.Fatal(err)
	}
	if roi.X != 560 || roi.Y != 400 || roi.W != 160 || roi.H != 80 || class != "mouth" {
		t.Fatalf("roi=%+v class=%q", roi, class)
	}
	if len(log) != 2 || !strings.Contains(log[1], "not applied as ROI2") {
		t.Fatalf("log=%v", log)
	}
	// Non-canonical part: ROI yes, region class left empty.
	if _, class, _ := roiFromSceneProposals(p, 12000, func(string) {}); class != "" {
		t.Fatalf("class=%q", class)
	}
}

func TestParseROI(t *testing.T) {
	if r, err := parseROI("1, 2,3,4"); err != nil || r.X != 1 || r.H != 4 {
		t.Fatalf("%+v %v", r, err)
	}
	for _, bad := range []string{"1,2,3", "a,2,3,4"} {
		if _, err := parseROI(bad); err == nil {
			t.Errorf("%q: want error", bad)
		}
	}
}
