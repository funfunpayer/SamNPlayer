# License system — concept (not built)

**Status:** design only. **Do not enforce** in shipping builds until the
owner flips an explicit “sharp” switch. Dev/test must keep working with
either no check or an **internal / invite** license baked into the app.

Related: proprietary app license (`LICENSE`), closed-source split
(`docs/CLOSED_SOURCE.md`), native scripts (`docs/SAMN_FORMAT.md`),
production plan (`docs/PRODUCTION_ROADMAP.md` stream E).

---

## Locked product decisions (owner)

| Decision | Choice |
|----------|--------|
| Retail key lifetime | **1 year** from issue (`exp = iat + 365 days`) |
| Seat | **One person** — one key named to that person (email / id). Honour system for v1 (file can be copied; no machine bind yet) |
| Internal / invite | **Built into the product path**: owner/invite uses the same verify UI as customers, but with `tier: "internal"` or `"invite"` and no expiry (or far-future `exp`). May also ship a **fixed invite key** for early testers that the owner issues once |
| When to build | **Not now.** When we start go-live: first a **license generator** (issuer CLI), then Settings import, then gates |
| Enforcement | Off until generator + Settings work and `.samn` is stable |

---

## Product rules (target behaviour)

| Mode | Generate | Playback |
|------|----------|----------|
| **Licensed** (valid ≤ 1 year from issue) | Full length; write `.samn` + `.funscript` | Play `.samn` and `.funscript` |
| **Internal / invite** | Same as licensed | Same as licensed |
| **Unlicensed / expired** | Cap at **1 minute** of output | Play **only normal `.funscript`**; refuse or degrade `.samn` / Neo-2 axes |

Intent: free/trial users can try the pipeline and use community scripts;
SamNPlayer-native extras (dual axes, contact bake, strength presets) sit
behind a **personal yearly key**. Invite/internal unlocks the same features
for the owner and invited testers without a shop purchase.

Clarifications (output rules):

1. **1-minute cap** = wall-clock of the *output script* (first 60 000 ms),
   not “one minute of tracking work”. Prefer output duration.
2. **Unlicensed playback of `.funscript`:** allowed even if SamNPlayer
   generated it (community format stays usable).
3. **Unlicensed `.samn`:** block Neo-2 play; still allow “Export .funscript”
   then play that.
4. **Offline:** keys must work without calling home every launch.

---

## Non-goals (for v1)

- No account server / DRM / always-online activation.
- No hardware dongle.
- No encrypting the whole `.samn` body (gates **app behaviour**, not file crypto).
- No multi-seat / household keys in v1 (one person per key).
- Not a substitute for going private + proprietary `LICENSE`.

---

## Developer / owner escape hatches (required before “scharf”)

Must ship **before** any enforcement lands:

| Hatch | Purpose |
|-------|---------|
| **Build tag / env `SAMN_LICENSE_OFF=1`** | Dev binaries: no checks at all |
| **Compile-time `license.Enforcement = false`** (default on `main` until go-live) | CI, clip7776, golden clips unchanged |
| **Internal / invite key** (`tier: internal` or `invite`, no expiry) | Owner + invited testers: same Settings path as customers |
| **`license.SkipInTests`** | Unit/frontend tests never need a key file |

Recommendation: default **enforcement off**. When ready, a release build
sets `Enforcement=true`. Day-to-day `main` and agent CI stay
unlicensed-friendly.

---

## License document (what a key contains)

Proposed signed payload (JSON claims inside a signed blob):

```json
{
  "v": 1,
  "sub": "person@email-or-stable-id",
  "iss": "funfunpayer",
  "iat": 1760000000,
  "exp": 1791536000,
  "tier": "standard",
  "seat": "person",
  "features": ["samn", "contact", "generate_full"],
  "note": "optional"
}
```

| Field | Meaning |
|-------|---------|
| `sub` | **One person** this key is issued to (email or stable id) |
| `seat` | Always `"person"` in v1 (documents the rule) |
| `exp` | End of validity (**issue + 1 year** for retail) |
| `tier` | `standard` (retail), `invite` (early tester), `internal` (owner) |
| `features` | Optional allow-list; v1 can treat all licensed equal |
| Invite / internal | `exp` omitted or far future + `tier: "invite"` / `"internal"` |

**File on disk (customer):** e.g. app data dir `license.key` or paste in
Settings. App verifies signature, then caches claims in memory for the
session.

---

## License generator (build when we start go-live)

Offline **issuer CLI** (owner machine only) — the first real code piece
when licensing starts:

```text
  license-tool issue \
      --sub "anna@example.com" \
      --years 1 \
      --tier standard \
      --key ~/.samn/issuer.ed25519 \
      --out Anna-2026.license.key

  # Invite / early tester (no yearly shop)
  license-tool issue \
      --sub "friend@…" \
      --tier invite \
      --exp never \
      --key ~/.samn/issuer.ed25519 \
      --out Friend-invite.license.key

  # Owner internal
  license-tool issue \
      --sub "owner" \
      --tier internal \
      --exp never \
      --key ~/.samn/issuer.ed25519 \
      --out Owner-internal.license.key
```

- Private signing key stays **outside** the git repo (password manager / USB).
- App embeds **public** verify key only.
- One command → one **person** key file → send out-of-band (email / shop).
- Customer: Settings → Import license → status shows name + “Valid until …”.

Store issuer private key **outside** the repo. Rotate by bumping `v` and
embedding a new public key (accept old `v` for a grace period).

---

## How to create licenses securely (options)

### A — Ed25519 signed files (recommended for v1)

- Strengths: offline, simple, no server, hard to forge without private key.
- Weaknesses: key file can be **copied** to another PC; not bound to machine.
  Acceptable for yearly **personal** license at this scale (honour system).

### B — Machine-bound (fingerprint)

- Defer unless abuse appears.

### C — Online activation

- Defer until there is real paid volume.

### D — Obfuscation-only / shared password

- Reject: trivial to crack once binary is distributed.

**Verdict:** start with **A** + the license generator above.

---

## Where enforcement would hook (when sharp)

| Gate | Unlicensed behaviour |
|------|----------------------|
| **Generate** (`GenerateScript` / CLI) | After build, truncate actions/`general` to `at ≤ 60_000`; still write both files; UI banner “Trial: 1 minute” |
| **Playback `.samn`** | Refuse start with clear English error + “Export .funscript to play without a license” |
| **Playback `.funscript`** | Allowed |
| **Bake axes / Save .samn extras** | Allowed only if licensed (optional at go-live) |
| **Update check** | Unrelated; do not couple |

UI: Settings → License → status (Valid until … / Trial / Invite) → paste or
browse key. Never log full key material.

---

## Risks and evaluation

| Risk | Severity | Mitigation |
|------|----------|------------|
| Cracked binary patches out check | High for any client app | Accept; closed source raises bar; don’t over-invest in DRM |
| Shared license files | Medium | One person + yearly renew; optional machine bind later |
| Leaked issuer private key | Critical | Offline only; revoke by new public key in next release |
| CI / clip7776 broken by enforcement | High | Default off; internal/invite; test skip |
| Confusing trial vs paid UX | Medium | One clear banner; English copy only |
| Truncate mid-stroke at 60s | Low | End on last keyframe ≤ 60s; document |

**Fit for SamNPlayer now:** good — binaries free to try, yearly **personal**
key for full native workflow; invite/internal for owner and early testers.
**Do not build** until go-live start; then **license generator first**,
then Settings, then gates behind `Enforcement=false`.

---

## Implementation phases (when we start)

1. **Docs only** ← this file (decisions locked).
2. **License generator** (`license-tool` / issuer CLI) — private, not on public site.
3. **Library stub** (`license` package): parse/verify, `Status()`; always
   `Licensed=true` while `Enforcement=false`.
4. **Settings UI** (import + status) — still no gates.
5. **Gates** behind flag: generate truncate + `.samn` play block.
6. **Release build** with `Enforcement=true` + retail + invite issue process.

---

## Remaining open decisions

1. Trial: may users **edit** `.samn` without a license, or only export
   funscript?
2. Renew: new key only, or grace days after `exp`?
3. Shop: manual issue via generator vs later Stripe webhook → auto issue?

**Settled:** one year; one person; internal/invite path; build generator
when we start; enforcement off until then.
