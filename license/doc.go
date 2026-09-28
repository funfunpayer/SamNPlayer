// Package license verifies SamNPlayer personal license keys (Ed25519).
//
// Enforcement is OFF by default (Enforcement=false). Gates must call
// EffectiveLicensed(), which returns true while enforcement is off so
// Generate/Play keep working. Feature gates call EffectiveHasFeature —
// same off-by-default pattern. Status() still reports the real key state
// so Settings and the license-tool can be exercised before go-live.
//
// Standard retail keys (€40 / year) stamp DefaultFeatures(), which includes
// virtual_person (Owner 2026-09-27: plugins in the standard license, not an
// addon). Virtual Person product is parked on tip; the feature id stays for
// key compatibility.
//
// See docs/LICENSE_SYSTEM.md, docs/PLUGIN_SYSTEM.md.
package license
