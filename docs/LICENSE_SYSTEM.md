# License system — concept (not built)

**Status:** design only. **Do not enforce** in shipping builds until the
owner flips an explicit “sharp” switch. Dev/test must keep working with
either no check or an infinite internal license.

Related: proprietary app license (`LICENSE`), closed-source split
(`docs/CLOSED_SOURCE.md`), native scripts (`docs/SAMN_FORMAT.md`).

---

## Product rules (target behaviour)

| Mode | Generate | Playback |
|------|----------|----------|
| **Licensed** (valid ≤ 1 year from issue) | Full length; write `.samn` + `.funscript` | Play `.samn` and `.funscript` |
| **Unlicensed / expired** | Cap at **1 minute** of output | Play **only normal `.funscript`**; refuse or degrade `.samn` / Neo-2 axes |

Intent: free/trial users can try the pipeline and use community scripts;
SamNPlayer-native extras (dual axes, contact bake, strength presets) sit
behind a paid yearly key.

Clarifications to decide before coding:

1. **1-minute cap** = wall-clock of the *output script* (first 60 000 ms),
   not “one minute of tracking work”. Prefer output duration.
2. **Unlicensed playback of `.funscript`:** allowed even if SamNPlayer
   generated it (community format stays usable).
3. **Unlicensed `.samn`:** block play (or play general-only as funscript
   export without vib/suc). Prefer **block Neo-2 play**, still allow
   “Export .funscript” then play that.
4. **Offline:** keys must work without calling home every launch (see
   security tradeoffs below).

---

## Non-goals (for v1)

- No account server / DRM / always-online activation (too heavy for this
  product stage).
- No hardware dongle.
- No encrypting the whole `.samn` body (breaks local edit and our own
  tooling). License gates **app behaviour**, not file crypto.
- Not a substitute for going private + proprietary `LICENSE`.

---

## Developer / owner escape hatches (required before “scharf”)

Must ship **before** any enforcement lands:

| Hatch | Purpose |
|-------|---------|
| **Build tag / env `SAMN_LICENSE_OFF=1`** | Dev binaries: no checks at all |
| **Compile-time `license.Enforcement = false`** (default on `main` until go-live) | CI, clip7776, golden clips unchanged |
| **Internal infinite key** | Owner machines: same UI path as customers, never expires |
| **`license.SkipInTests`** | Unit/frontend tests never need a key file |

Recommendation: default **enforcement off**. When ready, a single release
build sets `Enforcement=true` and ships only signed keys. Day-to-day
`main` and agent CI stay unlicensed-friendly.

---

## License document (what a key contains)

Proposed signed payload (JSON claims inside a signed blob):

```json
{
  "v": 1,
  "sub": "customer-or-email-id",
  "iss": "funfunpayer",
  "iat": 1760000000,
  "exp": 1791536000,
  "tier": "standard",
  "features": ["samn", "contact", "generate_full"],
  "note": "optional"
}
```

| Field | Meaning |
|-------|---------|
| `exp` | End of validity (**issue + 1 year** for retail) |
| `tier` | Future: `standard` / `pro` without new formats |
| `features` | Optional allow-list; v1 can treat all licensed equal |
| Infinite internal | `exp` omitted or far future (`9999-01-01`) + `tier: "internal"` |

**File on disk (customer):** e.g. `~/Library/.../SamNPlayer/license.key`
or paste in Settings. App verifies signature, then caches claims in
memory for the session.

---

## How to create licenses securely (options)

### A — Ed25519 signed files (recommended for v1)

- Owner keeps **private** signing key offline (password manager / USB).
- App embeds **public** verify key only.
- Issue script (owner machine only): reads claims → signs → writes
  `license.key` (base64 of `payload || signature`).
- Strengths: offline, simple, no server, hard to forge without private key.
- Weaknesses: key can be **copied** to another PC; not bound to machine.
  Acceptable for yearly personal license at this scale.

### B — Machine-bound (fingerprint)

- Key signs `exp` + hash of machine id (hostname + disk id, etc.).
- Stronger against casual sharing; painful for dual-boot / upgrades /
  support (“my PC died”).
- Defer unless abuse appears.

### C — Online activation

- Best revoke/abuse control; needs hosting, auth, offline grace, privacy.
- Defer until there is real paid volume.

### D — Obfuscation-only / shared password

- Reject: trivial to crack once binary is distributed.

**Verdict:** start with **A**. Document that licenses are portable by
design (one seat = honour system + yearly renew). Add B later if needed.

---

## Where enforcement would hook (when sharp)

| Gate | Unlicensed behaviour |
|------|----------------------|
| **Generate** (`GenerateScript` / CLI) | After build, truncate actions/`general` to `at ≤ 60_000`; still write both files; UI banner “Trial: 1 minute” |
| **Playback `.samn`** | Refuse start with clear English error + “Export .funscript to play without a license” |
| **Playback `.funscript`** | Allowed |
| **Bake axes / Save .samn extras** | Allowed only if licensed (optional; else trial users never see Neo-2 value—decide at go-live) |
| **Update check** | Unrelated; do not couple |

UI: Settings → License → status (Valid until … / Trial) → paste or
browse key. Never log full key material.

---

## Issuing workflow (owner, later)

```text
  [offline] license-tool issue \
      --sub "anna@…" --years 1 \
      --key ~/.samn/issuer.ed25519 \
      > Anna-2026.license.key

  Send file out-of-band (email / shop).
  Customer: Settings → Import license.
```

Internal:

```text
  license-tool issue --tier internal --exp never …
```

Store issuer private key **outside** the git repo. Rotate by bumping
`v` and embedding a new public key (accept old `v` for a grace period).

---

## Risks and evaluation

| Risk | Severity | Mitigation |
|------|----------|------------|
| Cracked binary patches out check | High for any client app | Accept; closed source raises bar; don’t over-invest in DRM |
| Shared license files | Medium | Yearly renew + optional machine bind later |
| Leaked issuer private key | Critical | Offline only; revoke by new public key in next release |
| CI / clip7776 broken by enforcement | High | Default off; infinite internal; test skip |
| Confusing trial vs paid UX | Medium | One clear banner; English copy only |
| Truncate mid-stroke at 60s | Low | End on last keyframe ≤ 60s; document |

**Fit for SamNPlayer now:** good — matches closed-source + public
downloads: binaries free to try, yearly key for full native workflow.
**Do not build** until after `.samn` lands and day-to-day testing is
stable; then implement behind `Enforcement=false`.

---

## Implementation phases (when approved)

1. **Docs only** ← this file.
2. **Library stub** (`license` package): parse/verify, `Status()`; always
   `Licensed=true` while `Enforcement=false`.
3. **Issuer CLI** (private repo or `tools/`, not in public site).
4. **Settings UI** (import + status) — still no gates.
5. **Gates** behind flag: generate truncate + `.samn` play block.
6. **Release build** with `Enforcement=true` + retail key process.

---

## Open decisions for the owner

1. Trial: may users **edit** `.samn` without a license, or only export
   funscript?
2. One license = one person or one household?
3. Renew: new key only, or grace days after `exp`?
4. Shop: manual issue vs later Stripe webhook → auto issue?

Until those are answered, keep enforcement **off** and keep testing with
full `.samn` locally.
