# Plugin system — host contract for Virtual Persons

**Status:** proposal (not implemented as a general host API yet)  
**Audience:** SamNPlayer maintainers + Animation Studio  
**Related:** experimental Go package [`virtualperson/`](../virtualperson/) on this branch · product work continues in [`funfunpayer/tracken`](https://github.com/funfunpayer/tracken) (Animation Studio)

---

## Goal

We are building a **Virtual Person** experience that should **attach to SamNPlayer** as a plugin / extension — not replace SamNPlayer’s generate/play/train core.

| Capability | Scope |
|------------|--------|
| Virtual Persons | Animated characters synced to playback (first: character-01; multi-character later) |
| Virtual props & activities | In-scene toys (MVP: give **dildo** → **titjob** activity), extensible catalog |
| Optional real devices | Sam Neo 2 first-class; Intiface/buttplug for other toys as a **parallel** output |
| Chat / persona (later) | Structured tags → props/activities; adults only; local LLM preferred |

**What this is not**

- Not moving the whole Animation Studio into this repository  
- Not a MultiFunPlayer-style C# `#:plugin` copy-paste  
- Not a marketplace / streaming-source plugin surface (see `docs/ROADMAP.md` — streaming plugins stay skipped)  
- Not deleting or renaming SamNPlayer

Today SamNPlayer already has **targeted** extensibility (Python analysis backends in `generator/backends.py`, Go packages, Wails bindings). It does **not** yet expose a first-class **plugin host** for UI + playback-tied Virtual Persons. That host API is what this document asks for.

---

## Why SamNPlayer needs a plugin / extension API

Virtual Persons need the **same clock** as video/funscript playback and optional access to the **same device path** the player already owns. Without a host contract, every overlay either forks the player or invents a second sync clock (jitter, double-drive on Neo 2, brittle GUI hacks).

A small, explicit API lets Animation Studio ship character/prop/activity logic **out of band** (tracken / later loadable module) while SamNPlayer remains the authoritative player and Sam Neo 2 owner.

---

## Proposed host APIs

Names are illustrative; implementation can be Go interfaces bound through Wails.

### 1. Load / lifecycle

| Host provides | Plugin uses |
|---------------|-------------|
| Discover / enable plugins (in-process Go module first; optional later: config list) | `Init(host Host) error` |
| Start / stop with app session | `Start(ctx)`, `Stop()` |
| Clean unregister of actions / UI | `Dispose()` — no leftover device writes |

**Requirement:** one writer policy for `device.Device` — either `player.Player` **or** the plugin’s ToyHub when “Virtual Person control” is armed, never both.

### 2. Playback clock sync

| Host provides | Plugin uses |
|---------------|-------------|
| Media / script time `NowMs() int64` | Sample funscript / drive activity phase |
| Play / pause / seek notifications | Reset or freeze animation |
| Optional tick aligned with player `OnFrame` (~20–50 Hz device cadence) | `Plugin.Tick(scriptPos, vibe, suck)` |

**Requirement:** plugins must not invent a second wall-clock for sync; they subscribe to the host clock.

### 3. Funscript / timeline hooks

| Host provides | Plugin uses |
|---------------|-------------|
| Current primary curve sample (0–100) and mapped vibe/suck | Motion bus / animation channels |
| Script load / clear / chapter change events | Switch activities or idle |
| Read-only access to `funscript.Script` / `.samn` metadata when loaded | Optional learning offline — not silent rewriting |

**Requirement:** plugins do **not** replace Generate’s classical writer path (`docs/AI_ADAPTER.md`). They may **read** playback state and optionally request device output under the ownership rule above.

### 4. UI surface

| Host provides | Plugin uses |
|---------------|-------------|
| A reserved panel / tab / overlay slot in the Wails frontend | Character stage, prop inventory, activity controls, optional chat |
| Event bridge host → JS (`EmitAnimation(pose)`, prop snapshots) | Sprite puppet / later Live2D |
| Settings section for plugin enable + Neo 2 sync toggle | Persist via existing settings store |

**Requirement:** Emotion look / English UI conventions (`docs/LANGUAGE.md`, `docs/GUI_MAKEUP.md`) apply to plugin chrome shipped inside SamNPlayer.

### 5. Device bridge (optional output)

| Host provides | Plugin uses |
|---------------|-------------|
| The live `device.Device` (BLE / Intiface / Mock) or a narrow `DeviceBridge` | Optional sync of activity intensity to Sam Neo 2 |
| Emergency stop / disconnect hooks | Hard cut on Stop / E-Stop |

**Requirement:** virtual props are the **primary** interaction metaphor; real toys are opt-in parallel output.

---

## Sketch (in-process Go)

```text
SamNPlayer (Wails host)
  player.Player ──clock──► PluginHost
  device.Device ──bridge─► PluginHost
  frontend slot  ◄─events─ PluginHost
                              │
                              ▼
                     Virtual Person plugin
                     (character registry, props,
                      activities, motion bus)
```

Minimal host surface (conceptual):

```go
type Host interface {
    NowMs() int64
    Device() DeviceBridge // may be nil
    EmitAnimation(pose PoseSample)
    // SubscribePlayState(fn) ...
}
```

The experimental [`virtualperson`](../virtualperson/) package on this PR sketches plugin-side types (`Plugin`, `PropInventory`, `ActivityCatalog`, `MotionBus`, `ToyHub`). It is a **foundation**, not a finished host.

---

## Ownership split

| Repo | Role |
|------|------|
| **SamNPlayer** (this repo) | Player, funscript/samn, Sam Neo 2, eventual **plugin host API**, thin integration |
| **tracken** → Animation Studio | Primary Virtual Person product code, assets, GUI experiments, MVP scenes |

Do not treat SamNPlayer#264 / `virtualperson/` as the long-term home for the full Animation Studio. Keep the host contract here so the player can load that work cleanly later.

---

## MVP acceptance (host side)

1. Documented host interface reviewed (this file).  
2. Wails can call `Start` / `Tick` / `Stop` on one registered Virtual Person plugin without breaking playback when the plugin is disabled.  
3. With plugin enabled and sync **off**, animation updates follow the playback clock; Neo 2 remains owned by `player.Player`.  
4. With sync **on** (explicit), ownership flips without double-drive (measured).  

Implementation of (2)–(4) is follow-up work; this PR establishes the **need** and the **contract**.

---

## References

- `device.Device`, `player.Player`, `funscript` — existing sync/device path  
- `docs/AI_ADAPTER.md` — AI proposes, classical measures  
- `docs/SAM_NEO_2_RESEARCH.md` — Neo 2 capabilities  
- `docs/ROADMAP.md` — streaming plugin surface remains out of scope  
- Animation Studio plan (project store): Virtual Person plugin plan / ecosystem research  
