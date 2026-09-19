package license

import "os"

// Enforcement gates Generate/Play when true. Default false — not sharp.
// Flip only for a deliberate go-live release build.
var Enforcement = false

// SkipInTests forces EffectiveLicensed true in unit/frontend test helpers.
var SkipInTests = false

// EffectiveLicensed is what Generate/Play must call.
// While Enforcement is false (or SAMN_LICENSE_OFF=1 / SkipInTests), always true.
func EffectiveLicensed(st Status) bool {
	if !Enforcement || SkipInTests || envLicenseOff() {
		return true
	}
	return st.Licensed
}

func envLicenseOff() bool {
	v := os.Getenv("SAMN_LICENSE_OFF")
	return v == "1" || v == "true" || v == "yes"
}
