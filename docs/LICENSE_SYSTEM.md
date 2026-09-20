# License system

**Status:** infrastructure landed, **enforcement OFF** (`license.Enforcement =
false`) — **developer mode through roadmap G0–G2**. Import/verify/Settings
work so we do not lose the path; Generate/Play are **not** limited yet.
Flip sharpness only after Generate + Neo 2 mapping feel stable
(`docs/PRODUCTION_ROADMAP.md` G4.E / E5).

Related: proprietary app license (`LICENSE`), closed-source split
(`docs/CLOSED_SOURCE.md`), native scripts (`docs/SAMN_FORMAT.md`),
production plan (`docs/PRODUCTION_ROADMAP.md` stream E).

---

## Locked product decisions (owner)

| Decision | Choice |
|----------|--------|
| Retail price | **€40** / year (EUR; one named seat) |
| Retail key lifetime | **1 year** from issue (`exp = iat + 365 days`) |
| Seat | **One person** — one key named to that person (email / id) |
| Internal / invite | Same Settings import; `tier: internal` or `invite`; no expiry |
| Generator | **`cmd/license-tool`** (offline Ed25519 issuer) — built |
| Library | **`license` package** — parse/verify/`Status`/`EffectiveLicensed` — built |
| Settings UI | Import paste/file + status + clear — built |
| Enforcement | **Off** until go-live flip |

---

## Product rules (when sharp)

| Mode | Generate | Playback |
|------|----------|----------|
| **Licensed** (≤ 1 year) | Full; `.samn` + `.funscript` | `.samn` + `.funscript` |
| **Internal / invite** | Same as licensed | Same as licensed |
| **Unlicensed / expired** | Cap **1 minute** output | `.funscript` only; block Neo-2 `.samn` play |

While enforcement is off, `EffectiveLicensed` is always true (full access).
`Status.Licensed` still reflects the real key so you can test import.

---

## What is built now

| Piece | Path | Notes |
|-------|------|-------|
| Claims + SNP1 token | `license/` | Ed25519, `SNP1.payload.sig` |
| Embedded public key | `license/pubkey.go` | Matches `license/testdata/issuer.ed25519` (**DEV**) |
| Issuer CLI | `cmd/license-tool` | `genkey`, `issue`, `verify` |
| GUI API | `cmd/gui-wails/app_license.go` | `GetLicenseStatus`, `ImportLicense*`, `ClearLicense`, `LicenseAllowsFullFeatures` |
| Settings section | `frontend/src/settings.js` | License card (English) |

### Issue a key (dev issuer)

```bash
go run ./cmd/license-tool issue \
  --key license/testdata/issuer.ed25519 \
  --sub you@example.com --years 1 \
  --out /tmp/you.key

go run ./cmd/license-tool issue \
  --key license/testdata/issuer.ed25519 \
  --sub owner --tier internal --exp never \
  --out /tmp/owner.key

go run ./cmd/license-tool verify --file /tmp/owner.key
```

Then: Settings → License → Import from file / paste.

### Escape hatches

| Hatch | Purpose |
|-------|---------|
| `license.Enforcement = false` (default) | Not sharp |
| `SAMN_LICENSE_OFF=1` | Force full access even if sharp |
| `license.SkipInTests` | Tests |
| Internal/invite key | Owner + early testers |

---

## Non-goals (v1)

- No account server / always-online activation
- No machine bind (honour system for one-person seat)
- No `.samn` body encryption

---

## Where gates will hook (later)

| Gate | Unlicensed behaviour |
|------|----------------------|
| Generate | Truncate output to ≤ 60 000 ms; banner “Trial: 1 minute” |
| Playback `.samn` | Refuse + “Export .funscript…” |
| Playback `.funscript` | Allowed |

Call site already prepared: `App.LicenseAllowsFullFeatures()` →
`license.EffectiveLicensed(...)`.

---

## Before paid go-live

1. `license-tool genkey --out ~/.samn/issuer` (private offline)
2. Replace `EmbeddedPublicKeyHex` with the new public key
3. Stop using `license/testdata/issuer.ed25519` for real customers
4. Flip `Enforcement=true` in a dedicated release build
5. Wire Generate/Play to `LicenseAllowsFullFeatures`

---

## Remaining open decisions

1. Trial: edit `.samn` without a license, or only export funscript?
2. Renew: new key only, or grace days after `exp`?
3. Shop: manual issue vs later Stripe → auto issue?
