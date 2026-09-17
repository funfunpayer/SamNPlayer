package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/funfunpayer/SamNPlayer/generator"
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
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s generate --video FILE --roi x,y,w,h [--output FILE] [options]\n", os.Args[0])
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *video == "" || *roiStr == "" {
		fs.Usage()
		return 2
	}
	parts := strings.Split(*roiStr, ",")
	if len(parts) != 4 {
		fmt.Fprintln(os.Stderr, "roi must be x,y,w,h")
		return 2
	}
	var roi generator.ROI
	for i, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			fmt.Fprintf(os.Stderr, "roi: %v\n", err)
			return 2
		}
		switch i {
		case 0:
			roi.X = v
		case 1:
			roi.Y = v
		case 2:
			roi.W = v
		case 3:
			roi.H = v
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
	}
	err := generator.GenerateWithContext(context.Background(), *video, roi, out, opts,
		func(line string) { fmt.Fprintln(os.Stderr, line) },
		nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate:", err)
		return 1
	}
	fmt.Println(out)
	return 0
}
