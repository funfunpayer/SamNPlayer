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
| Standard features | `samn`, `contact`, `generate_full`, **`virtual_person`** (plugins **in** the €40 key — not an addon; Owner 2026-09-27) |
| Internal / invite | Same Settings import; `tier: internal` or `invite`; no expiry; same feature set |
| Generator | **`cmd/license-tool`** (offline Ed25519 issuer) — built |
| Library | **`license` package** — parse/verify/`Status`/`EffectiveLicensed`/`EffectiveHasFeature` — built |
| Settings UI | Import paste/file + status + clear — built; Virtual Person host H1 — built |
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

**Gates are wired** in `GenerateScript` (sets `MaxOutputMs=60000` when not
effective) and `StartPlayback` (refuses `.samn` when not effective). Flipping
`Enforcement=true` therefore activates trial/Play rules without further code.

---

## What is built now

| Piece | Path | Notes |
|-------|------|-------|
| Claims + SNP1 token | `license/` | Ed25519, `SNP1.payload.sig` |
| Feature ids | `license/features.go` | `DefaultFeatures()` includes `virtual_person` |
| Feature gate | `EffectiveHasFeature` | Same off-by-default pattern as `EffectiveLicensed` |
| Embedded public key | `license/pubkey.go` | Matches `license/testdata/issuer.ed25519` (**DEV**) |
| Issuer CLI | `cmd/license-tool` | `genkey`, `issue`, `verify` (prints features) |
| GUI API | `cmd/gui-wails/app_license.go` | `GetLicenseStatus`, `ImportLicense*`, `ClearLicense`, `LicenseAllowsFullFeatures` |
| Plugin host H1 | `pluginhost/` + `app_pluginhost.go` | Drop-folder discover/install only; VP product parked |
| Settings section | `frontend/src/settings.js` | License card + Plugins (Install / Open folder; no VP Enable) |

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
| Plugin pack feature (parked) | N/A while product parked — feature id reserved |

Call sites:

- `App.LicenseAllowsFullFeatures()` → `license.EffectiveLicensed(...)`  
- Future plugin host: `license.EffectiveHasFeature(..., FeatureVirtualPerson)`  

While Enforcement is off, feature gates return true. Virtual Person product
Enable UI is scrubbed from tip (Rel35); re-wire when unparked.

---

## Before paid go-live

1. `license-tool genkey --out ~/.samn/issuer` (private offline)
2. Replace `EmbeddedPublicKeyHex` with the new public key
3. Stop using `license/testdata/issuer.ed25519` for real customers
4. Flip `Enforcement=true` in a dedicated release build
5. Wire Generate/Play to `LicenseAllowsFullFeatures` (already wired)
6. When Virtual Person returns: gate Enable behind `EffectiveHasFeature(..., FeatureVirtualPerson)`

---

## Remaining open decisions

1. Trial: edit `.samn` without a license, or only export funscript?
2. Renew: new key only, or grace days after `exp`?
3. Shop: manual issue vs later Stripe → auto issue?
