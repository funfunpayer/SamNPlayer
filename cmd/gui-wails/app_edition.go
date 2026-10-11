package main

import (
	"errors"

	"github.com/funfunpayer/SamNPlayer/edition"
)

// errStudioOnly is returned by AI-training and learning bindings in the
// customer build. The bindings stay so the Wails JS API is the same in both
// editions; the customer GUI never shows the controls that call them.
var errStudioOnly = errors.New("not available in this edition — AI training and learning tools are part of the internal studio build")

// IsStudioBuild tells the GUI which edition is running (see package edition).
func (a *App) IsStudioBuild() bool { return edition.Studio() }

func requireStudio() error {
	if edition.Studio() {
		return nil
	}
	return errStudioOnly
}
