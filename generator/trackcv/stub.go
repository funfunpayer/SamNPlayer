//go:build !opencv || !cgo || windows

// Package trackcv is the Go CSRT tracker. This stub file is compiled when
// the binary is built without `-tags opencv` (the default), so a fresh
// clone without libopencv-dev still builds. The real implementation lives
// behind `//go:build cgo && opencv && !windows` — enable it in CI/release
// with `-tags opencv` after installing system OpenCV.
package trackcv
