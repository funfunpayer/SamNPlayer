//go:build studio

package generator

import (
	"os"
	"path/filepath"
	"strings"
)

// copyStudioScripts copies the studio-only training scripts into the temp
// directory used by writeScriptToTemp so that training paths can find them
// next to generate_funscript.py. No-op / absent in the customer build.
func copyStudioScripts(dir string) error {
	entries, err := studioPythonFiles.ReadDir("studio")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, "_test.py") {
			continue
		}
		content, err := studioPythonFiles.ReadFile("studio/" + name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, name), content, 0644); err != nil {
			return err
		}
	}
	return nil
}
