package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/funfunpayer/SamNPlayer/license"
	"github.com/funfunpayer/SamNPlayer/logging"
)

// LicenseStatus is the Settings / API view of the current license.
// Enforcement is off by default — Effective stays true until go-live.
type LicenseStatus = license.Status

// GetLicenseStatus reads license.key from the config dir and evaluates it.
func (a *App) GetLicenseStatus() LicenseStatus {
	path, err := license.DefaultPath()
	if err != nil {
		return LicenseStatus{
			Enforcement: license.Enforcement,
			State:       "none",
			Message:     "Could not resolve config dir.",
			Error:       err.Error(),
			Effective:   license.EffectiveLicensed(license.Status{}),
		}
	}
	tok, err := license.LoadFile(path)
	if err != nil {
		st := license.Evaluate("", license.EmbeddedPublicKey, time.Now().UTC())
		st.Error = err.Error()
		st.Message = "Could not read license file."
		return st
	}
	return license.Evaluate(tok, license.EmbeddedPublicKey, time.Now().UTC())
}

// ImportLicenseText verifies a pasted token and saves it as license.key.
func (a *App) ImportLicenseText(token string) (LicenseStatus, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return LicenseStatus{}, fmt.Errorf("empty license text")
	}
	c, err := license.ParseAndVerify(token, license.EmbeddedPublicKey)
	if err != nil {
		return LicenseStatus{}, fmt.Errorf("invalid license: %w", err)
	}
	if c.Expired(time.Now().UTC()) {
		return LicenseStatus{}, fmt.Errorf("license expired on %s", c.ValidUntil())
	}
	path, err := license.DefaultPath()
	if err != nil {
		return LicenseStatus{}, err
	}
	if err := license.SaveToken(path, token); err != nil {
		return LicenseStatus{}, err
	}
	logging.Info("license imported", "sub", c.Sub, "tier", c.Tier)
	return a.GetLicenseStatus(), nil
}

// ImportLicenseFile opens a file picker, then imports the selected key file.
func (a *App) ImportLicenseFile() (LicenseStatus, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import license key",
		Filters: []runtime.FileFilter{
			{DisplayName: "License key", Pattern: "*.key;*.license;*.txt"},
			{DisplayName: "All files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return LicenseStatus{}, err
	}
	if path == "" {
		return a.GetLicenseStatus(), nil
	}
	tok, err := license.LoadFile(path)
	if err != nil {
		return LicenseStatus{}, err
	}
	if tok == "" {
		return LicenseStatus{}, fmt.Errorf("file is empty")
	}
	return a.ImportLicenseText(tok)
}

// ClearLicense removes the imported license.key (back to no key).
func (a *App) ClearLicense() (LicenseStatus, error) {
	path, err := license.DefaultPath()
	if err != nil {
		return LicenseStatus{}, err
	}
	if err := license.RemoveFile(path); err != nil {
		return LicenseStatus{}, err
	}
	return a.GetLicenseStatus(), nil
}

// LicenseAllowsFullFeatures reports whether Generate/Play should run without
// trial limits. While license.Enforcement is false, this is always true.
func (a *App) LicenseAllowsFullFeatures() bool {
	return license.EffectiveLicensed(a.GetLicenseStatus())
}
