// Package license verifies SamNPlayer personal license keys (Ed25519).
//
// Enforcement is OFF by default (Enforcement=false). Gates must call
// EffectiveLicensed(), which returns true while enforcement is off so
// Generate/Play keep working. Status() still reports the real key state
// so Settings and the license-tool can be exercised before go-live.
//
// See docs/LICENSE_SYSTEM.md.
package license
