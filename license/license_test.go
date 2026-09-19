package license

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testIssuer(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	priv, err := LoadPrivateKey(filepath.Join("testdata", "issuer.ed25519"))
	if err != nil {
		t.Fatal(err)
	}
	return priv.Public().(ed25519.PublicKey), priv
}

func TestIssueVerifyRoundTrip(t *testing.T) {
	pub, priv := testIssuer(t)
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	tok, err := Issue(priv, NewStandardClaims("anna@example.com", 1, now))
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParseAndVerify(tok, pub)
	if err != nil {
		t.Fatal(err)
	}
	if c.Sub != "anna@example.com" || c.Tier != TierStandard || c.Seat != SeatPerson {
		t.Fatalf("claims %+v", c)
	}
	if c.NeverExpires() || c.Expired(now) {
		t.Fatalf("exp issue: never=%v expired=%v until=%s", c.NeverExpires(), c.Expired(now), c.ValidUntil())
	}
	if c.Expired(now.AddDate(1, 0, 1)) != true {
		t.Fatal("should be expired after 1 year + 1 day")
	}
}

func TestInviteNeverExpires(t *testing.T) {
	pub, priv := testIssuer(t)
	now := time.Now().UTC()
	tok, err := Issue(priv, NewInviteClaims("owner", TierInternal, now))
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParseAndVerify(tok, pub)
	if err != nil {
		t.Fatal(err)
	}
	if !c.NeverExpires() || c.Expired(now.AddDate(50, 0, 0)) {
		t.Fatalf("internal should never expire")
	}
	st := Evaluate(tok, pub, now)
	if !st.Licensed || st.State != "internal" {
		t.Fatalf("status %+v", st)
	}
}

func TestBadSignature(t *testing.T) {
	_, priv := testIssuer(t)
	tok, err := Issue(priv, NewStandardClaims("x", 1, time.Now()))
	if err != nil {
		t.Fatal(err)
	}
	otherPub, _, _ := GenerateKeyPair()
	if _, err := ParseAndVerify(tok, otherPub); err == nil {
		t.Fatal("expected signature error")
	}
}

func TestEffectiveLicensedWhileOff(t *testing.T) {
	old := Enforcement
	Enforcement = false
	defer func() { Enforcement = old }()
	st := Evaluate("", EmbeddedPublicKey, time.Now())
	if !EffectiveLicensed(st) {
		t.Fatal("enforcement off must allow")
	}
	if st.Licensed {
		t.Fatal("no key should not be Licensed")
	}
}

func TestEffectiveLicensedWhenSharp(t *testing.T) {
	old := Enforcement
	Enforcement = true
	defer func() { Enforcement = old }()
	t.Setenv("SAMN_LICENSE_OFF", "")
	st := Evaluate("", EmbeddedPublicKey, time.Now())
	if EffectiveLicensed(st) {
		t.Fatal("enforcement on without key must deny")
	}
	pub, priv := testIssuer(t)
	tok, _ := Issue(priv, NewStandardClaims("a", 1, time.Now()))
	st = Evaluate(tok, pub, time.Now())
	if !EffectiveLicensed(st) {
		t.Fatal("valid key must allow when sharp")
	}
}

func TestEnvLicenseOff(t *testing.T) {
	old := Enforcement
	Enforcement = true
	defer func() { Enforcement = old }()
	t.Setenv("SAMN_LICENSE_OFF", "1")
	st := Evaluate("", EmbeddedPublicKey, time.Now())
	if !EffectiveLicensed(st) {
		t.Fatal("SAMN_LICENSE_OFF must allow")
	}
}

func TestSaveLoadFile(t *testing.T) {
	pub, priv := testIssuer(t)
	tok, err := Issue(priv, NewInviteClaims("t", TierInvite, time.Now()))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "license.key")
	if err := SaveToken(path, tok); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFile(path)
	if err != nil || got != tok {
		t.Fatalf("load %q err=%v", got, err)
	}
	st := Evaluate(got, pub, time.Now())
	if !st.Licensed || st.State != "invite" {
		t.Fatalf("%+v", st)
	}
	_ = os.Remove(path)
}
