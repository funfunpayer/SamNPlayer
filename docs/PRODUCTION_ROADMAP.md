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
| **v0.5.8** | Tagged | Playback polish prior |
| **v0.5.9** | **Shipped** | `.samn`, portable ffmpeg, English GUI lock, license infra (not sharp) |
| **v0.5.10** | This train (local green; tag after CI billing fixed) | Bugfix + GUI motion + multi body-part regions + soft masks / N distance partners |

**Owner steps after this PR merges to `main`:**

1. Confirm CI green on `main` (or merge with known billing-blocked empty jobs)  
2. Spot-check GUI (Play + Generate) in English — **no popup on startup**  
3. Prefer **portable** zip (ffmpeg included). Settings → “What you need” should match reality  
4. AI Train (optional): mark → Use for training → train; empty dataset must warn, not traceback  
5. Tag `v0.5.10` and push the tag → release workflow builds Win/Linux + portable archives  
6. Run `./scripts/sync-public-site.sh` then `./scripts/publish-public-release.sh v0.5.10`  
7. Prefer **portable** download on the public site  
8. Hardware: one BLE or Intiface play session if a device is available  

### Owner pre-release checklist (before tagging)

| Check | Pass? |
|-------|-------|
| Startup: no update/error popup; Log quiet if already on latest | |
| Play: load `.samn` / video, seek, heat map | |
| Generate: mark ROI → generate → script opens in Play | |
| Tf/Tj: tip + Fix ROI2 / +Target / +Mask if you use them | |
| Settings → Check folders: ffmpeg found (portable) | |
| AI Train skipped OR samples collected before train | |
| Portable zip from previous tag still runs on a clean Windows folder | |

Agent can prepare the PR and version bump; **tagging on `main` is the
release switch** (`CONTRIBUTING.md`).

---

## Open workstreams (split by “bit”)

Priority order within each stream. **Target** = earliest sensible ship once
prerequisites are met — not a calendar promise.

**Next focus (parallel OK):** A (script quality) **and** C (GUI improve).
Quality still wins if a GUI change would hurt Generate/Play.

### A — Script quality (highest product priority)

| ID | Item | Target | Prerequisites | Your test |
|----|------|--------|---------------|-----------|
| A1 | Golden-clip manifest + FunGen r | next after 0.5.9 | Your clips + FunGen refs | Run Bench tab; paste report |
| A2 | Tf/Tj + AI/auto two-ROI bake-off | after A1 | Two-body golden | Manual vs suggested ROI scores |
| A2b | **Multi body-part regions** (Face…Vagina; fixed/tracked/mask) | with A2 | Taxonomy + AI train 9 marks | Mark 9 classes; Tf/Tj Fix ROI2; preferred CSV |
| A2c | **Soft masks + >2 distance partners** (min tip→N targets) | after A2b | CLI `--target`/`--mask`, GUI +Target/+Mask | Extra target changes stroke; mask punches cam features |
| A3 | Go `auto_roi` (if equal to Python) | after A1 | Clip gate `SELF_BUILD` | Same ROI as Python on 3 clips |
| A4 | Contact-vibration “feel” | when device free | Real Neo 2 | Subjective note + log |

### B — Desktop platforms & tools

| ID | Item | Target | Prerequisites | Your test |
|----|------|--------|---------------|-----------|
| B1 | Portable zip on real Win PC | with 0.5.9 | Release assets | Fresh PC, no system ffmpeg |
| B2 | Install video tools (Settings) | with 0.5.9 | Network allow | Missing ffmpeg → install → Generate |
| B3 | macOS GUI build | later | Mac runner + notarization | Open `.app`, Play script-alone |
| B4 | Linux WebKitGTK smoke | with 0.5.9 | Linux box | Play + Generate CSRT |

### C — Player / GUI improve (back on the plan — competitive)

Bar: `docs/COMPETITIVE.md`. English only. Play stays home. No card-grid
redesign / no purple theme. Polish that helps Play + Generate first.

| ID | Item | Target | Prerequisites | Your test |
|----|------|--------|---------------|-----------|
| C1 | Language sweep (leftover DE) | ongoing | — | Every tab; screenshot any DE |
| C2 | Thanks / feedback English | done (0.5.9) | — | usable / borderline / unusable |
| C3 | README vs public portal layout | ongoing | SamNPlayer-site | Same brand + portable CTA |
| C5 | **GUI improve pass** (spacing, hierarchy, Play first viewport, Generate clarity) | **next** | Your notes / screenshots | Before/after screenshots; Play + Generate still work |
| C6 | Settings License block polish (status readable, import clear) | with C5 | E3 | Import key → status obvious |
| C4 | Curve zoom / BPM grid | after C5 | Design call | Optional polish |
| C7 | Empty states / errors (Make playable, no script, missing ffmpeg) | with C5 | — | Trigger each banner; English + clear CTA |

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

## What still needs **your** testing

1. **Private golden clips** — Bench / FunGen numbers (A1)  
2. **Real Neo 2** — contact vibration + Extended-O feel (A4, F4)  
3. **Windows portable zip** without system ffmpeg (B1)  
4. **GUI improve (C5)** — note what feels cluttered / unclear in Play + Generate; screenshots help  
5. **Public site** vs README (C3)  
6. **License path** — `license-tool` → Settings → License (E2–E3; not sharp)  
7. **Any DE leftover** (C1)

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
- License enforcement (E5 — infra already on main)  
- WebGL video sharpen / soft upscale  
- Full pure-Go H.264  
- Go ports of flow/`grid_lk` without golden win  
- Wails-v3 / frameless / card redesign (rejected — see `ROADMAP.md`)  
