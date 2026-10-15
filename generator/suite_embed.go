//go:build suite

package generator

import "embed"

//go:embed suite_scripts/*
var suitePythonFiles embed.FS
