# Plugin system — host contract for Virtual Persons

**Status:** H0 landed — host contract + license gate (no full Animation Studio)  
**Audience:** SamNPlayer maintainers + Animation Studio  
**Related:** package [`pluginhost/`](../pluginhost/) · license feature `virtual_person` · draft [#264](https://github.com/funfunpayer/SamNPlayer/pull/264) (parked full `virtualperson/` scaffold; do not force-merge)

---

## Goal

Virtual Persons **attach to SamNPlayer** as a plugin / extension — they do not
replace Generate / Play / Train.

| Capability | Scope |
|------------|--------|
| Virtual Persons | Animated characters synced to playback |
| Virtual props & activities | In-scene toys (later MVP: give dildo → titjob) |
| Optional real devices | Sam Neo 2 / Intiface as **parallel** output |
| Chat / persona (later) | Structured tags; local LLM preferred |

**What this is not**

- Not moving Animation Studio into this repository  
- Not a MultiFunPlayer C# plugin copy  
- Not a marketplace  
- Not Python analysis backends (`generator/backends.py`) — separate surface  

---

## License (Owner 2026-09-27)

| Decision | Choice |
|----------|--------|
| Feature id | **`virtual_person`** |
| Monetization | **Included in the standard €40 / year key** — not a separate addon |
| Stamped with | `samn`, `contact`, `generate_full`, `virtual_person` (`license.DefaultFeatures`) |
| Gate | `license.EffectiveHasFeature(st, FeatureVirtualPerson)` → `App.LicenseAllowsVirtualPerson` / `EnableVirtualPersonHost` |
| Enforcement off (current) | Feature gate is open (dev); Everyday CSRT unchanged |
| Enforcement on (later) | Host enable requires a valid key listing `virtual_person` |

Do **not** flip `license.Enforcement = true` without Owner OK.

---

## Why a host API

Virtual Persons need the **same clock** as video/funscript playback and optional
access to the device path the player already owns. Without a host contract,
overlays fork the player or invent a second sync clock.

---

## H0 acceptance (this slice)

1. Documented host interface (this file).  
2. `pluginhost.Slot` — Enable / Disable / Tick (no-op when disabled).  
3. License feature stamped + `HasFeature` / `EffectiveHasFeature` evaluated.  
4. Settings → Virtual Person host status + Enable/Disable (GUI load rule).  
5. Everyday Create / Play / CSRT unchanged; Enforcement stays off.

**Not in H0:** full `virtualperson/` package from #264, sprite overlay, ToyHub
device ownership flip, OnFrame tick wiring into playback.

---

## Proposed host APIs (later stages)

### 1. Load / lifecycle

Discover / enable → `Init` / `Start` / `Stop` / `Dispose`. One writer for
`device.Device` — either `player.Player` **or** plugin ToyHub when armed.

### 2. Playback clock sync

`NowMs()`, play/pause/seek, optional tick aligned with player `OnFrame`.

### 3. Funscript / timeline hooks

Read-only curve sample + script load events. Plugins do **not** replace
Generate’s classical writer (`docs/AI_ADAPTER.md`).

### 4. UI surface

Reserved panel / overlay slot; Emotion look + English UI conventions.

### 5. Device bridge (optional)

Live `device.Device` or narrow bridge; E-Stop on Stop. Virtual props remain
the primary metaphor; real toys are opt-in.

### Sketch

```text
SamNPlayer (Wails host)
  player.Player ──clock──► pluginhost.Slot
  device.Device ──bridge─► (later)
  frontend slot  ◄─events─ (later)
                              │
                              ▼
                     Virtual Person plugin
```

Minimal surface today (`pluginhost`):

```go
type Host interface {
    NowMs() int64
    EmitAnimation(pose PoseSample)
}
```

---

## Ownership split

| Repo | Role |
|------|------|
| **SamNPlayer** | Player, funscript/samn, Neo 2, **plugin host API**, thin integration |
| **tracken** / Animation Studio | Character/prop/activity product code, assets |

Keep the host contract here so Animation Studio can attach later. Prefer
cherry-picking from #264 over force-merging the dirty draft.

---

## References

- `license/` — `FeatureVirtualPerson`, `DefaultFeatures`, `EffectiveHasFeature`  
- `docs/LICENSE_SYSTEM.md` — product rules + enforcement  
- `device.Device`, `player.Player`, `funscript` — existing sync path  
- `docs/AI_ADAPTER.md` — AI proposes, classical measures  
