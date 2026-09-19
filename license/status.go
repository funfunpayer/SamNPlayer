package license

import (
	"crypto/ed25519"
	"fmt"
	"os"
	"strings"
	"time"
)

// Status is the UI/API view of the current license file.
type Status struct {
	// Present: a license.key file exists (may still be invalid).
	Present bool `json:"present"`
	// Licensed: signature OK and not expired (real key state).
	Licensed bool `json:"licensed"`
	// Effective: what gates should use (true while Enforcement is off).
	Effective bool `json:"effective"`
	// Enforcement mirrors the package flag for the UI.
	Enforcement bool   `json:"enforcement"`
	State       string `json:"state"` // none | valid | expired | invalid | invite | internal
	Sub         string `json:"sub,omitempty"`
	Tier        string `json:"tier,omitempty"`
	ValidUntil  string `json:"validUntil,omitempty"`
	Message     string `json:"message"`
	Error       string `json:"error,omitempty"`
}

// Evaluate builds Status from an optional token and verify key.
func Evaluate(token string, pub ed25519.PublicKey, now time.Time) Status {
	st := Status{
		Enforcement: Enforcement,
		Message:     "No license imported — trial rules apply only when enforcement is on.",
		State:       "none",
	}
	token = strings.TrimSpace(token)
	if token == "" {
		st.Effective = EffectiveLicensed(st)
		if !Enforcement {
			st.Message = "No license imported. Enforcement is off — full features available."
		}
		return st
	}
	st.Present = true
	c, err := ParseAndVerify(token, pub)
	if err != nil {
		st.State = "invalid"
		st.Error = err.Error()
		st.Message = "License file present but not valid."
		st.Effective = EffectiveLicensed(st)
		return st
	}
	st.Sub = c.Sub
	st.Tier = c.Tier
	st.ValidUntil = c.ValidUntil()
	if c.Expired(now) {
		st.State = "expired"
		st.Message = fmt.Sprintf("License for %s expired on %s.", c.Sub, c.ValidUntil())
		st.Effective = EffectiveLicensed(st)
		return st
	}
	st.Licensed = true
	switch c.Tier {
	case TierInternal:
		st.State = "internal"
		st.Message = fmt.Sprintf("Internal license for %s (no expiry).", c.Sub)
	case TierInvite:
		st.State = "invite"
		st.Message = fmt.Sprintf("Invite license for %s (no expiry).", c.Sub)
	default:
		st.State = "valid"
		st.Message = fmt.Sprintf("Licensed for %s until %s.", c.Sub, c.ValidUntil())
	}
	st.Effective = EffectiveLicensed(st)
	if !Enforcement {
		st.Message += " Enforcement is off."
	}
	return st
}

// LoadFile reads a license token from path (empty path → empty token).
func LoadFile(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
