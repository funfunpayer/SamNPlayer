package player

import "time"

// ExtendedOOptions steuert den Extended-O-Klimax-Modus: reduziert
// Vibration und Sog kurzzeitig auf ein Minimum, hält, und stellt danach
// (optional sanft) das vorherige Niveau wieder her. Semantik orientiert
// sich an der "Extended O"-Funktion des echten Geräts bzw. am
// ExtendedO-Tool aus Kyure-A/mcp-svakom-samneo.
type ExtendedOOptions struct {
	// MinLevel ist die Intensität, auf die während des Haltens reduziert wird.
	MinLevel float64
	// HoldDuration ist die Zeit, die auf MinLevel gehalten wird.
	HoldDuration time.Duration
	// RestoreDuration ist die Zeit, um von MinLevel zurück auf das vorherige
	// Niveau zu rampen. 0 = sofort (kein Ramping).
	RestoreDuration time.Duration
}

// DefaultExtendedOOptions liefert plausible Startwerte, angelehnt an das
// Referenzprojekt (minimumLevel 0.1, holdDuration 10s, restoreDuration 500ms).
func DefaultExtendedOOptions() ExtendedOOptions {
	return ExtendedOOptions{
		MinLevel:        0.1,
		HoldDuration:    10 * time.Second,
		RestoreDuration: 500 * time.Millisecond,
	}
}

// TriggerExtendedO fordert einen Extended-O-Zyklus an. Nicht-blockierend:
// wird während der laufenden Wiedergabe (Play) beim nächsten Tick
// abgearbeitet. Ist bereits ein Trigger anhängig, wird der neue verworfen,
// statt sich aufzustauen.
func (p *Player) TriggerExtendedO(opts ExtendedOOptions) {
	select {
	case p.extendedOCh <- opts:
	default:
		// Es wartet schon ein Trigger auf Abarbeitung - zusätzliche
		// Tastendrücke in der Zwischenzeit einfach ignorieren.
	}
}
