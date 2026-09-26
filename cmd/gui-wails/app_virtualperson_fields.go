package main

import (
	"sync"
	"sync/atomic"

	"github.com/funfunpayer/SamNPlayer/virtualperson"
)

// Virtual Person host state (see app_virtualperson.go).
// Kept in a separate file so the main App struct stays readable; Go allows
// embedding extra fields only by extending the type — we use package-level
// helpers on *App via these fields added through a thin extension pattern.
//
// Note: App is defined in app.go. To add fields we patch app.go in the same
// commit. This file only documents the contract and holds shared helpers if
// needed later.

// vpFields are the members that belong on App for the plugin host.
// They are declared on App in app.go (search: vpPlugin).
type vpFields struct {
	vpOnce    sync.Once
	vpMu      sync.Mutex
	vpPlugin  *virtualperson.Plugin
	vpHost    *AppHost
	vpClockMs atomic.Int64
}
