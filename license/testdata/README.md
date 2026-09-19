# Dev issuer keypair (NOT for paid go-live)

`issuer.ed25519` matches `license.EmbeddedPublicKeyHex` in `pubkey.go`.

Use only for local/CI tests and early invite keys while enforcement is off:

```bash
go run ./cmd/license-tool issue \
  --key license/testdata/issuer.ed25519 \
  --sub owner --tier internal --exp never \
  --out /tmp/owner.key
```

Before selling licenses: run `license-tool genkey`, keep the private key
offline, replace `EmbeddedPublicKeyHex`, and stop using this folder.
