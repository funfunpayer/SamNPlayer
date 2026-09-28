package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/funfunpayer/SamNPlayer/funscript"
	"github.com/funfunpayer/SamNPlayer/generator"
	"github.com/funfunpayer/SamNPlayer/generator/bodyparts"
	"github.com/funfunpayer/SamNPlayer/sam"
)

// runGenerate is the headless generate path for release testing
// (same GenerateWithContext as the GUI — Go auto / Python fallback).
func runGenerate(args []string) int {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	video := fs.String("video", "", "video path (required)")
	output := fs.String("output", "", "output .funscript path (default: next to video)")
	roiStr := fs.String("roi", "", "x,y,w,h in pixels (required)")
	backend := fs.String("backend", "csrt", "tracking backend (csrt = Go-eligible)")
	preferPython := fs.Bool("prefer-python", false, "force Python even when Go path is eligible")
	maxFrames := fs.Int("max-frames", 0, "limit frames (0 = all)")
	autoRetry := fs.Bool("auto-retry", true, "signal-param auto-retry")
	rhythmGrid := fs.Bool("rhythm-grid", false, "stroke signal from the most rhythmic flow cell near the box (drift-robust, Go CSRT only)")
	contactPoints := fs.String("contact-points", "", "contact_points.py JSON: teachers' contact points steer the rhythm grid where the box is out of reach (needs --rhythm-grid)")
	contactMinAgree := fs.Int("contact-min-agree", 0, "keep only contact points at least this many teachers agreed on")
	sceneProposals := fs.String("scene-proposals", "", "scene_roles.py <clip>.scene.json: opt-in, use the proposed primary target as --roi (and its body part as region class) when --roi is not given; the partner is only logged, never applied as ROI2")
	sceneAtMs := fs.Int64("scene-at-ms", 0, "video time (ms) whose scene proposal --scene-proposals uses")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s generate --video FILE (--roi x,y,w,h | --scene-proposals FILE) [--output FILE] [options]\n", os.Args[0])
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *video == "" || (*roiStr == "" && *sceneProposals == "") {
		fs.Usage()
		return 2
	}
	var roi generator.ROI
	regionClass := ""
	if *roiStr != "" {
		var err error
		if roi, err = parseROI(*roiStr); err != nil {
			fmt.Fprintln(os.Stderr, "roi:", err)
			return 2
		}
	} else {
		var err error
		if roi, regionClass, err = roiFromSceneProposals(*sceneProposals, *sceneAtMs, func(line string) { fmt.Fprintln(os.Stderr, line) }); err != nil {
			fmt.Fprintln(os.Stderr, "scene proposals:", err)
			return 2
		}
	}
	out := *output
	if out == "" {
		ext := filepath.Ext(*video)
		out = strings.TrimSuffix(*video, ext) + ".funscript"
	}
	opts := generator.Options{
		Backend:      *backend,
		PreferPython: *preferPython,
		MaxFrames:    *maxFrames,
		AutoRetry:    *autoRetry,
		RhythmGrid:   *rhythmGrid,
		RegionClass:  regionClass,

		ContactPointsFile:     *contactPoints,
		ContactPointsMinAgree: *contactMinAgree,
	}
	err := generator.GenerateWithContext(context.Background(), *video, roi, out, opts,
		func(line string) { fmt.Fprintln(os.Stderr, line) },
		nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate:", err)
		return 1
	}
	fmt.Println(out)
	if script, err := funscript.Load(out); err == nil && funscript.IsDistanceProfile(script.Metadata.Profile) {
		if err := sam.WriteEnrichedSidecar(out, script); err != nil {
			fmt.Fprintln(os.Stderr, "sam sidecar:", err)
		} else {
			fmt.Fprintln(os.Stderr, "sam:", sam.SidecarPath(out))
		}
	}
	return 0
}

func parseROI(s string) (generator.ROI, error) {
	var roi generator.ROI
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return roi, fmt.Errorf("must be x,y,w,h")
	}
	v := make([]int, 4)
	for i, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return roi, err
		}
		v[i] = n
	}
	return generator.ROI{X: v[0], Y: v[1], W: v[2], H: v[3]}, nil
}

// roiFromSceneProposals is the explicit opt-in: the user passed the file,
// so the proposed primary target becomes the ROI. Everything applied is
// logged; the contact partner is only reported (no silent ROI2).
func roiFromSceneProposals(path string, atMs int64, log func(string)) (generator.ROI, string, error) {
	s, err := generator.LoadSceneProposals(path)
	if err != nil {
		return generator.ROI{}, "", err
	}
	p, ok := s.At(atMs)
	if !ok {
		return generator.ROI{}, "", fmt.Errorf("%s has no proposal", path)
	}
	c := p.Primary
	roi := generator.ROI{X: c.X, Y: c.Y, W: c.W, H: c.H}
	class := ""
	if bodyparts.IsCanonical(c.Class) {
		class = c.Class
	}
	log(fmt.Sprintf("scene proposals: ROI %d,%d,%d,%d = %s (scene %q, confidence %.2f, window %d-%d ms) region class %q",
		roi.X, roi.Y, roi.W, roi.H, c.Class, p.SceneType, p.Confidence, p.StartMs, p.EndMs, class))
	if p.Partner != nil {
		log(fmt.Sprintf("scene proposals: contact partner %s at %d,%d,%d,%d - proposal only, not applied as ROI2",
			p.Partner.Class, p.Partner.X, p.Partner.Y, p.Partner.W, p.Partner.H))
	}
	return roi, class, nil
}
