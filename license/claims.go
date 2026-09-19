package license

import "time"

// Claims is the signed payload inside a license key file.
type Claims struct {
	V        int      `json:"v"`
	Sub      string   `json:"sub"`
	Iss      string   `json:"iss"`
	Iat      int64    `json:"iat"`
	Exp      *int64   `json:"exp,omitempty"` // nil = never expires (invite/internal)
	Tier     string   `json:"tier"`          // standard | invite | internal
	Seat     string   `json:"seat"`          // person
	Features []string `json:"features,omitempty"`
	Note     string   `json:"note,omitempty"`
}

const (
	TierStandard  = "standard"
	TierInvite    = "invite"
	TierInternal  = "internal"
	SeatPerson    = "person"
	IssuerName    = "funfunpayer"
	ClaimsVersion = 1
)

// NeverExpires reports invite/internal keys without exp.
func (c Claims) NeverExpires() bool {
	return c.Exp == nil
}

// Expired reports whether exp is in the past (never-expiring keys are false).
func (c Claims) Expired(now time.Time) bool {
	if c.Exp == nil {
		return false
	}
	return now.Unix() > *c.Exp
}

// ValidUntil returns a human date or "never".
func (c Claims) ValidUntil() string {
	if c.Exp == nil {
		return "never"
	}
	return time.Unix(*c.Exp, 0).UTC().Format("2006-01-02")
}
