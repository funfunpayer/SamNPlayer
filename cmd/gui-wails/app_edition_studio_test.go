//go:build studio

package main

import "testing"

func TestStudioBuildFlag(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if !NewApp().IsStudioBuild() {
		t.Fatal("-tags studio must report the studio edition")
	}
}
