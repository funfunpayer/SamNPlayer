package funscript

import "strings"

// Profile-Kürzel. tf und tj sind zwei Namen für dasselbe Rezept:
// Abstand zweier Regionen, auf dem Neo 2 nur der Sog (Position),
// Vibration bleibt aus.
const (
	ProfileStandard = "standard"
	ProfileWeich    = "weich"
	ProfileTF       = "tf"
	ProfileTJ       = "tj"
)

// NormalizeProfile macht aus Schreibweisen einen kanonischen Namen.
func NormalizeProfile(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "tf", "tj":
		return ProfileTJ
	case "weich":
		return ProfileWeich
	case "", "standard":
		return ProfileStandard
	default:
		return strings.ToLower(strings.TrimSpace(name))
	}
}

// IsDistanceProfile: Signal ist der Abstand zweier Regionen, Sog führt.
func IsDistanceProfile(name string) bool {
	n := NormalizeProfile(name)
	return n == ProfileTJ || n == ProfileTF
}

// DeviceRecipe ist das, was in die Funscript-Metadata geschrieben wird,
// damit die Wiedergabe ohne extra Klick denselben Aktor trifft.
type DeviceRecipe struct {
	Sync       string  `json:"sync"`
	MinSuction float64 `json:"min_suction"`
	TickMs     int64   `json:"tick_ms"`
	MaxSpeed   float64 `json:"max_speed"`
	Smoothing  float64 `json:"smoothing"`
}

// RecipeFor liefert MapOptions für ein Profil.
// tf/tj: Sog folgt der Position (Kompression), Vibration = 0,
// leichter Sog-Boden damit der Kontakt nicht abreisst.
func RecipeFor(name string) MapOptions {
	opts := DefaultMapOptions()
	switch NormalizeProfile(name) {
	case ProfileTJ, ProfileTF:
		opts.Sync = SyncSuctionPosition
		opts.MinVibration = 0
		opts.MinSuction = 0.20
		opts.TickMs = 50
		opts.MaxSpeed = 0.50
		opts.Smoothing = 0.22
	case ProfileWeich:
		opts.Sync = SyncIndependent
		opts.MinVibration = 0.12
		opts.MinSuction = 0.10
		opts.Smoothing = 0.28
	}
	return opts
}

// RecipeMeta ist die JSON-Seite von RecipeFor.
func RecipeMeta(name string) DeviceRecipe {
	o := RecipeFor(name)
	return DeviceRecipe{
		Sync:       o.Sync.String(),
		MinSuction: o.MinSuction,
		TickMs:     o.TickMs,
		MaxSpeed:   o.MaxSpeed,
		Smoothing:  o.Smoothing,
	}
}

// ApplyRecipe füllt nur Felder, die der Aufrufer nicht gesetzt hat.
func ApplyRecipe(base MapOptions, name string, syncSet bool) MapOptions {
	if !IsDistanceProfile(name) && NormalizeProfile(name) != ProfileWeich {
		return base
	}
	r := RecipeFor(name)
	if !syncSet {
		base.Sync = r.Sync
	}
	if base.MinSuction == 0 && r.MinSuction > 0 {
		base.MinSuction = r.MinSuction
	}
	return base
}
