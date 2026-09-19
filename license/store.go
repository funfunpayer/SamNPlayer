package license

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultFileName is the license file under the app config dir.
const DefaultFileName = "license.key"

// DefaultPath returns <UserConfigDir>/SamNPlayer/license.key.
func DefaultPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "SamNPlayer", DefaultFileName), nil
}

// SaveToken writes token to path (creates parent dirs).
func SaveToken(path, token string) error {
	if path == "" {
		return fmt.Errorf("license: empty path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(token)+"\n"), 0o600)
}

// RemoveFile deletes the license file if present.
func RemoveFile(path string) error {
	if path == "" {
		return nil
	}
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
