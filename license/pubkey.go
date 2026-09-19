package license

import "encoding/hex"

// EmbeddedPublicKeyHex is the Ed25519 public key baked into the app.
// Matching private key for local/dev issue: license/testdata/issuer.ed25519
// (DEV ONLY — replace both before any paid go-live; see docs/LICENSE_SYSTEM.md).
const EmbeddedPublicKeyHex = "a8e458a4bba35682b248eab8d1e615e3f6aef417b4c8b470462882522f0ba345"

// EmbeddedPublicKey is the decoded verify key. Panics only if the constant is corrupt.
var EmbeddedPublicKey = mustDecodePub(EmbeddedPublicKeyHex)

func mustDecodePub(h string) []byte {
	b, err := hex.DecodeString(h)
	if err != nil || len(b) != 32 {
		panic("license: bad EmbeddedPublicKeyHex")
	}
	return b
}
