package license

// Feature IDs stamped on retail / invite / internal keys.
//
// Owner decision (2026-09-27): Virtual Person / plugin host is included in
// the standard €40 license — not a separate addon. Feature id: virtual_person.
const (
	FeatureSamn          = "samn"
	FeatureContact       = "contact"
	FeatureGenerateFull  = "generate_full"
	FeatureVirtualPerson = "virtual_person"
)

// DefaultFeatures is stamped on every new standard / invite / internal key.
func DefaultFeatures() []string {
	return []string{
		FeatureSamn,
		FeatureContact,
		FeatureGenerateFull,
		FeatureVirtualPerson,
	}
}

// HasFeature reports whether claims list the given feature id.
func (c Claims) HasFeature(id string) bool {
	for _, f := range c.Features {
		if f == id {
			return true
		}
	}
	return false
}

// EffectiveHasFeature is what plugin-host / Virtual Person gates must call.
//
// While Enforcement is false (or SAMN_LICENSE_OFF=1 / SkipInTests), always
// true — same pattern as EffectiveLicensed, so Everyday CSRT and current
// developer builds stay open. When Enforcement is on: requires a valid
// (Licensed) key that lists the feature.
func EffectiveHasFeature(st Status, feature string) bool {
	if !Enforcement || SkipInTests || envLicenseOff() {
		return true
	}
	if !st.Licensed {
		return false
	}
	for _, f := range st.Features {
		if f == feature {
			return true
		}
	}
	return false
}
