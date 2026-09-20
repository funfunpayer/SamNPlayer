package main

import (
	"fmt"

	"github.com/funfunpayer/SamNPlayer/update"
)

type UpdateCheckResult struct {
	Available bool            `json:"available"`
	Release   *update.Release `json:"release,omitempty"`
	AssetName string          `json:"assetName,omitempty"`
	Error     string          `json:"error,omitempty"`
}

func (a *App) CheckForUpdate() UpdateCheckResult {
	rel, err := update.CheckLatest()
	if err != nil {
		// No public release yet → "up to date", not a toast/popup-worthy error.
		if err == update.ErrNoRelease {
			return UpdateCheckResult{Available: false}
		}
		return UpdateCheckResult{Error: err.Error()}
	}
	if !update.IsNewer(update.Version, rel.TagName) {
		return UpdateCheckResult{Available: false}
	}
	asset, ok := rel.AssetForThisPlatform("gui")
	if !ok {
		return UpdateCheckResult{Available: true, Release: rel, Error: "no matching binary for this platform in the release"}
	}
	return UpdateCheckResult{Available: true, Release: rel, AssetName: asset.Name}
}

// ApplyUpdate lädt das zur aktuellen Plattform passende Asset herunter
// (mit Prüfsummen-Verifikation, siehe update.Download) und startet das
// Programm damit neu. Kehrt im Erfolgsfall nicht zurück (Prozessende).
func (a *App) ApplyUpdate() error {
	rel, err := update.CheckLatest()
	if err != nil {
		return err
	}
	asset, ok := rel.AssetForThisPlatform("gui")
	if !ok {
		return fmt.Errorf("no matching binary for this platform in the release")
	}
	checksum, checksumErr := rel.ChecksumFor(asset.Name)
	if checksumErr != nil {
		checksum = "" // Download() protokolliert und läuft ohne Verifikation weiter
	}
	path, err := update.Download(asset, checksum)
	if err != nil {
		return err
	}
	return update.ApplyAndRestart(path)
}

func (a *App) CurrentVersion() string {
	return update.Describe()
}
