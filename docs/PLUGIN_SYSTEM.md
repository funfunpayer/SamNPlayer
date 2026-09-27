# Plugin system — host contract for Virtual Persons

**Status:** H1 landed — OnFrame tick wiring + `virtualperson/` core (props/activities/bus)  
**Audience:** SamNPlayer maintainers + Animation Studio  
**Related:** package [`pluginhost/`](../pluginhost/) · [`virtualperson/`](../virtualperson/) · license feature `virtual_person` · draft [#264](https://github.com/funfunpayer/SamNPlayer/pull/264) (parked full dump; cherry-pick only)

---

## Goal

Virtual Persons **attach to SamNPlayer** as a plugin / extension — they do not
replace Generate / Play / Train.

| Capability | Scope |
|------------|--------|
| Virtual Persons | Animated characters synced to playback |
| Virtual props & activities | In-scene toys (MVP: give dildo → titjob) |
| Optional real devices | Sam Neo 2 / Intiface as **parallel** output (ToyHub; sync **off**) |
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

## H1 acceptance (this slice)

1. Documented host interface (this file).  
2. `pluginhost.Slot` — Enable / Disable / Tick (no-op when disabled).  
3. License feature stamped + `HasFeature` / `EffectiveHasFeature` evaluated.  
4. Settings → Virtual Person host status + Enable/Disable + Give dildo / Start titjob.  
5. **OnFrame tick wiring** — `StartPlayback` → `player.OnFrame` → `tickVirtualPersonHost` when enabled.  
6. **`virtualperson/` core** cherry-picked from #264: props, activities, motion bus, passthrough control, ToyHub (sync default **off**).  
7. Pose events emitted as `virtualperson:pose` (no sprite overlay UI yet).  
8. Everyday Create / Play / CSRT unchanged; Enforcement stays off.

**Deferred (follow-up):** sprite / puppet overlay UI; ToyHub device-ownership flip
(player remains sole writer); full chat GUI; force-merge of dirty #264.

---

## Host + plugin sketch

```text
SamNPlayer (Wails host)
  player.Player ──OnFrame──► App.tickVirtualPersonHost
                               ├─ pluginhost.Slot.Tick (clock / count)
                               └─ virtualperson.Plugin.Tick (bus / activity)
  device.Device ──bridge─► ToyHub (sync OFF — follow-up)
  frontend        ◄─events─ virtualperson:pose (overlay later)
```

```go
// pluginhost
type Host interface {
    NowMs() int64
    EmitAnimation(pose PoseSample)
}

// virtualperson (richer; implemented by App)
type Host interface {
    Device() DeviceBridge
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
