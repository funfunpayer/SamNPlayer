// Package generator erzeugt .funscript-Dateien aus Video per klassischem
// CV-Motion-Tracking (kein Deep Learning, kein trainiertes Modell - der
// Nutzer markiert eine Bildregion, ein OpenCV-Tracker verfolgt sie).
package generator

import (
	"fmt"
	"strconv"
)

type ROI struct {
	X, Y, W, H int
}

type Options struct {
	Profile string
	ROI2    *ROI
}

func buildArgs(scriptPath, videoPath, outputPath string, roi ROI, opts Options) []string {
	args := []string{scriptPath, "--video", videoPath, "--output", outputPath,
		"--roi", fmt.Sprintf("%d,%d,%d,%d", roi.X, roi.Y, roi.W, roi.H)}
	if opts.Profile == "weich" || opts.Profile == "tf" || opts.Profile == "tj" {
		args = append(args, "--profile", opts.Profile)
	}
	if opts.ROI2 != nil {
		args = append(args, "--roi2",
			fmt.Sprintf("%d,%d,%d,%d", opts.ROI2.X, opts.ROI2.Y, opts.ROI2.W, opts.ROI2.H))
	}
	_ = strconv.Itoa
	return args
}

func BuildArgsForTest(opts Options) []string {
	return buildArgs("script.py", "v.mp4", "o.funscript", ROI{}, opts)
}
