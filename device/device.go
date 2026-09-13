// Package device kapselt die Ansteuerung eines Toys hinter einem generischen
// Interface, damit der Player nicht wissen muss, ob am anderen Ende ein
// echtes BLE-Gerät oder ein Mock für Tests hängt.
package device

import "context"

// Device ist die minimale Steuerschnittstelle, die der Player benötigt.
// Intensitäten sind immer 0.0 (aus) bis 1.0 (voll).
type Device interface {
	// Connect baut die Verbindung zum Gerät auf (z.B. BLE-Scan + Connect).
	Connect(ctx context.Context) error
	// Disconnect trennt die Verbindung sauber.
	Disconnect() error
	// SetVibration setzt die Vibrationsintensität.
	SetVibration(intensity float64) error
	// SetSuction setzt die Sog-/Saugintensität.
	SetSuction(intensity float64) error
	// Stop schaltet beide Kanäle sofort aus (z.B. bei Pause/Abbruch).
	Stop() error
}
