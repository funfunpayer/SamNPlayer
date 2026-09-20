# Production roadmap (rock-solid)

Living plan for what ships next, what **you** (owner) must test, and how
results turn into fixes. Product UI/docs stay **English** (simple English
where possible). We talk in German; the app does not.

Related: `docs/ROADMAP.md` (engineering checklist), `docs/SELF_BUILD.md`
(quality gate), `docs/PLATFORMS.md`, `docs/COMPETITIVE.md`,
`docs/LANGUAGE.md`.

---

## How you test → how we improve

```text
  You run a check (table below)
       ↓
  Note: clip name, settings, what you saw, pass/fail
       ↓
  Put results in: GitHub issue OR message on this agent with
      - clip id / path (not the private video itself if licensed)
      - expected vs actual
      - screenshot of English UI if layout issue
       ↓
  We: reproduce → fix on a branch → clip/CI gate → merge → next tag
```

**Where results go:** prefer a GitHub issue titled `test: <area> — <clip>`  
so history stays searchable. Attach screenshots (English UI). Do not commit
private golden videos into the repo (`docs/GOLDEN_CLIPS.md`).

**What we then build:** only changes that keep or improve **script quality**
(Quality Doctor + your usable/borderline/unusable feedback). Layout/docs
fixes are fine if they do not dilute Generate/Play.

---

## Release train (now)

| Version | Status | Contents (short) |
|---------|--------|------------------|
| **v0.5.8** | Already tagged | Playback polish prior |
| **v0.5.9** | Tagged | `.samn` SoT, Go audio-check, ROI verify, portable ffmpeg, lean probe, platforms stake, English GUI lock |
| **v0.5.10** | This train (local green; tag after CI billing fixed) | Bugfix + GUI motion + multi body-part regions (Face…Vagina, Fix ROI2) |

**Owner steps after this PR merges to `main`:**

1. Confirm CI green on `main`  
2. Spot-check GUI (Play + Generate) in English  
3. Tag `v0.5.9` and push the tag → release workflow builds Win/Linux + portable archives  
4. Prefer **portable** download on the public site (ffmpeg included)  
5. Hardware: one BLE or Intiface play session if a device is available  

Agent can prepare the PR and version bump; **tagging on `main` is the
release switch** (`CONTRIBUTING.md`).

---

## Open workstreams (split by “bit”)

Priority order within each stream. **Target** = earliest sensible ship once
prerequisites are met — not a calendar promise.

### A — Script quality (highest product priority)

| ID | Item | Target | Prerequisites | Your test |
|----|------|--------|---------------|-----------|
| A1 | Golden-clip manifest + FunGen r | next after 0.5.9 | Your clips + FunGen refs | Run Bench tab; paste report |
| A2 | Tf/Tj + AI/auto two-ROI bake-off | after A1 | Two-body golden | Manual vs suggested ROI scores |
| A2b | **Multi body-part regions** (Face…Vagina; fixed/tracked/mask) | with A2 | Taxonomy + AI train 9 marks | Mark 9 classes; Tf/Tj Fix ROI2; preferred CSV |
| A3 | Go `auto_roi` (if equal to Python) | after A1 | Clip gate `SELF_BUILD` | Same ROI as Python on 3 clips |
| A4 | Contact-vibration “feel” | when device free | Real Neo 2 | Subjective note + log |

### B — Desktop platforms & tools

| ID | Item | Target | Prerequisites | Your test |
|----|------|--------|---------------|-----------|
| B1 | Portable zip on real Win PC | with 0.5.9 | Release assets | Fresh PC, no system ffmpeg |
| B2 | Install video tools (Settings) | with 0.5.9 | Network allow | Missing ffmpeg → install → Generate |
| B3 | macOS GUI build | later | Mac runner + notarization | Open `.app`, Play script-alone |
| B4 | Linux WebKitGTK smoke | with 0.5.9 | Linux box | Play + Generate CSRT |

### C — Player / GUI (English, competitive)

| ID | Item | Target | Prerequisites | Your test |
|----|------|--------|---------------|-----------|
| C1 | Language sweep (no leftover DE UI) | 0.5.9 | — | Click every tab; note any leftover DE |
| C2 | Thanks / feedback buttons English | 0.5.9 (done) | — | After Generate → usable/… |
| C3 | README vs public portal layout | ongoing | SamNPlayer-site | Both show same brand + portable CTA |
| C4 | Curve zoom / BPM grid | later | Design call | Optional polish |

### D — Mobile player (no generator)

| ID | Item | Target | Prerequisites | Your test |
|----|------|--------|---------------|-----------|
| D1 | Scope freeze (`PLATFORMS.md`) | done (doc) | — | Read & confirm |
| D2 | Shared Go core API sketch | later | D1 | Review package list |
| D3 | iOS/Android prototype | later | BLE APIs + signing | Play `.samn` to device |

### E — License / distribution

| ID | Item | Target | Prerequisites | Your test |
|----|------|--------|---------------|-----------|
| E1 | License concept (1 year, 1 person, invite/internal) | done | — | Read `LICENSE_SYSTEM.md` |
| E2 | **License generator** (`cmd/license-tool`) | done (not sharp) | Dev issuer in testdata | Issue standard + internal; `verify` |
| E3 | Settings import + status | done (not sharp) | E2 | Settings → License: import / clear / status |
| E4 | Library `EffectiveLicensed` + escape hatches | done (enforcement off) | — | `go test ./license/` |
| E5 | Enforcement (trial 1 min / `.samn` play) | later / go-live | Replace pubkey; owner OK | Trial vs licensed vs invite |

### F — Stability & bugfix (every release)

| ID | Item | Target | Prerequisites | Your test |
|----|------|--------|---------------|-----------|
| F1 | clip7776 playback suite | every tag | Fixture in tree | CI + local `pb_clip7776_test.py` |
| F2 | Cancel generate mid-run | every tag | Short clip | Cancel → UI recovers |
| F3 | Make playable (HEVC/mkv) | every tag | Sample file | Banner → convert → play |
| F4 | Device connect test toggle | every tag | Device or Mock | Mock play full script |

---

## What still needs **your** testing (before calling 0.5.9 “rock solid”)

1. **Private golden clips** — Bench / FunGen numbers (A1)  
2. **Real Neo 2** — contact vibration + Extended-O feel (A4, F4)  
3. **Windows portable zip** on a machine without ffmpeg (B1)  
4. **Public site** copy vs README — same English CTA (C3)  
5. **Any DE leftover** you spot in the GUI (C1) — send screenshot  
6. **License path (not sharp)** — issue with `license-tool`, import in Settings → License (E2–E3)

Agent-side before tag: CI green, clip7776 suite, version sync, English
user-string bugfix on this branch.

---

## Split rule (quality first)

Work may be split across branches/PRs by stream (A/B/C…).  
**Do not** ship a stream if it makes generated scripts worse.  
Self-build only when equal or better (`docs/SELF_BUILD.md`).

## Language lock

| Surface | Language |
|---------|----------|
| GUI, help, status, screenshots | English (simple) |
| README / docs / CHANGELOG / portal | English |
| Code comments (new) | English preferred |
| Chat with owner | German |

---

## Not in this release (parked → next plan)

- macOS notarized build  
- Mobile player binary  
- License enforcement (generator + Settings first — see stream E)
- WebGL video sharpen / soft upscale  
- Full pure-Go H.264  
- Go ports of flow/`grid_lk` without golden win  
