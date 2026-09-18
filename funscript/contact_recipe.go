package funscript

import (
	"encoding/json"
	"fmt"
	"os"
)

// SaveContactRecipe schreibt Kontakt-Vibration-Einstellungen in die
// device_recipe-Metadata eines bestehenden .funscript - analog zu
// SaveOMarkers: nur metadata.device_recipe wird aktualisiert, übrige
// Metadata und Actions bleiben erhalten.
func SaveContactRecipe(path string, enabled bool, span float64, curve string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("funscript: Datei konnte nicht gelesen werden: %w", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("funscript: ungültiges JSON: %w", err)
	}

	meta := map[string]json.RawMessage{}
	if raw, ok := doc["metadata"]; ok && len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &meta); err != nil {
			return fmt.Errorf("funscript: metadata ungültig: %w", err)
		}
	}

	recipe := DeviceRecipe{}
	if raw, ok := meta["device_recipe"]; ok && len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &recipe); err != nil {
			return fmt.Errorf("funscript: device_recipe ungültig: %w", err)
		}
	}
	// Sync fehlt oft bei älteren Skripten - Tf/Tj-Kontakt braucht suction_position.
	if recipe.Sync == "" {
		recipe.Sync = SyncSuctionPosition.String()
	}
	recipe.ContactVibration = enabled
	if enabled {
		recipe.ContactVibrationSpan = EffectiveContactSpan(span)
		recipe.ContactVibrationCurve = NormalizeContactCurve(curve)
	} else {
		recipe.ContactVibrationSpan = 0
		recipe.ContactVibrationCurve = ""
	}

	recipeJSON, err := json.Marshal(recipe)
	if err != nil {
		return err
	}
	meta["device_recipe"] = recipeJSON
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	doc["metadata"] = metaJSON

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}
