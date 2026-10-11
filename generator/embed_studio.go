//go:build studio

package generator

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed studio/*.py studio/requirements-ai-train.txt
var studioPythonFiles embed.FS

func extractStudioScripts(dir string) error {
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

func readStudioFile(name string) ([]byte, error) {
	return studioPythonFiles.ReadFile("studio/" + name)
}
