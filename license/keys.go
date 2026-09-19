package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"os"
)

// GenerateKeyPair returns a new Ed25519 private (64) and public (32) key.
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

// LoadPrivateKey reads a raw 64-byte Ed25519 private key from path.
func LoadPrivateKey(path string) (ed25519.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) == ed25519.PrivateKeySize {
		return ed25519.PrivateKey(b), nil
	}
	// PEM without parsing libs: reject — we only support raw for v1.
	return nil, fmt.Errorf("license: private key file must be raw %d bytes (got %d)", ed25519.PrivateKeySize, len(b))
}

// LoadPublicKey reads a raw 32-byte Ed25519 public key from path.
func LoadPublicKey(path string) (ed25519.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) == ed25519.PublicKeySize {
		return ed25519.PublicKey(b), nil
	}
	return nil, fmt.Errorf("license: public key file must be raw %d bytes (got %d)", ed25519.PublicKeySize, len(b))
}

// WriteKeyPair writes raw priv/pub files (0600 / 0644).
func WriteKeyPair(privPath, pubPath string, pub ed25519.PublicKey, priv ed25519.PrivateKey) error {
	if err := os.WriteFile(privPath, priv, 0o600); err != nil {
		return err
	}
	if pubPath == "" {
		pubPath = privPath + ".pub"
	}
	return os.WriteFile(pubPath, pub, 0o644)
}
