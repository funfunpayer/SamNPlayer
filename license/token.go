package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const tokenPrefix = "SNP1"

// Issue signs claims with priv (64-byte ed25519 private key) and returns the token string.
func Issue(priv ed25519.PrivateKey, c Claims) (string, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("license: private key must be %d bytes", ed25519.PrivateKeySize)
	}
	if err := normalizeClaims(&c); err != nil {
		return "", err
	}
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(priv, payload)
	return tokenPrefix + "." +
		base64.RawURLEncoding.EncodeToString(payload) + "." +
		base64.RawURLEncoding.EncodeToString(sig), nil
}

// ParseAndVerify checks the token signature against pub and returns claims.
func ParseAndVerify(token string, pub ed25519.PublicKey) (Claims, error) {
	var zero Claims
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || parts[0] != tokenPrefix {
		return zero, fmt.Errorf("license: bad token format (want SNP1.payload.sig)")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return zero, fmt.Errorf("license: bad payload encoding: %w", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return zero, fmt.Errorf("license: bad signature encoding: %w", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return zero, fmt.Errorf("license: public key must be %d bytes", ed25519.PublicKeySize)
	}
	if !ed25519.Verify(pub, payload, sig) {
		return zero, fmt.Errorf("license: signature invalid")
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return zero, fmt.Errorf("license: bad claims JSON: %w", err)
	}
	if c.V != ClaimsVersion {
		return zero, fmt.Errorf("license: unsupported version %d", c.V)
	}
	return c, nil
}

func normalizeClaims(c *Claims) error {
	if c.Sub == "" {
		return fmt.Errorf("license: sub (person id) required")
	}
	if c.V == 0 {
		c.V = ClaimsVersion
	}
	if c.Iss == "" {
		c.Iss = IssuerName
	}
	if c.Iat == 0 {
		c.Iat = time.Now().Unix()
	}
	if c.Seat == "" {
		c.Seat = SeatPerson
	}
	if c.Seat != SeatPerson {
		return fmt.Errorf("license: seat must be %q in v1", SeatPerson)
	}
	switch c.Tier {
	case "":
		c.Tier = TierStandard
	case TierStandard, TierInvite, TierInternal:
	default:
		return fmt.Errorf("license: unknown tier %q", c.Tier)
	}
	return nil
}

// NewStandardClaims builds a 1-year personal retail key.
func NewStandardClaims(sub string, years int, now time.Time) Claims {
	if years < 1 {
		years = 1
	}
	exp := now.AddDate(years, 0, 0).Unix()
	return Claims{
		V:        ClaimsVersion,
		Sub:      sub,
		Iss:      IssuerName,
		Iat:      now.Unix(),
		Exp:      &exp,
		Tier:     TierStandard,
		Seat:     SeatPerson,
		Features: []string{"samn", "contact", "generate_full"},
	}
}

// NewInviteClaims builds a non-expiring invite or internal key.
func NewInviteClaims(sub, tier string, now time.Time) Claims {
	if tier != TierInvite && tier != TierInternal {
		tier = TierInvite
	}
	return Claims{
		V:        ClaimsVersion,
		Sub:      sub,
		Iss:      IssuerName,
		Iat:      now.Unix(),
		Exp:      nil,
		Tier:     tier,
		Seat:     SeatPerson,
		Features: []string{"samn", "contact", "generate_full"},
	}
}
