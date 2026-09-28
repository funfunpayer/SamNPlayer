# Plugin system — host contract for Virtual Persons

**Status:** H1 landed — drop-folder install + OnFrame tick + `virtualperson/` core  
**Audience:** SamNPlayer maintainers + Animation Studio  
**Related:** package [`pluginhost/`](../pluginhost/) · [`virtualperson/`](../virtualperson/) · license feature `virtual_person` · draft [#264](https://github.com/funfunpayer/SamNPlayer/pull/264) (parked full dump; cherry-pick only) · Animation Studio [`funfunpayer/tracken`](https://github.com/funfunpayer/tracken)

**End users (German):** [`PLUGIN_INSTALL_DE.md`](PLUGIN_INSTALL_DE.md)

---

## End-user load path (H1 — keep this simple)

1. Drop the **Virtual Person** pack folder into the SamNPlayer **Plugins** folder  
   (`%APPDATA%\SamNPlayer\plugins` on Windows), **or** Settings → Virtual Person → **Install pack…**
2. Settings → Virtual Person → **Enable**
3. Done — no CLI for end users

The pack must contain `samn-plugin.json` at its root. Refresh discovers it; Enable arms the host slot and starts the in-process scene bus.

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
- Not dynamic Go `.so` loading for end users (asset pack + host APIs)  
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

## Package layout + version handshake

### `samn-plugin.json` (required)

```json
{
  "apiVersion": 1,
  "id": "virtual_person",
  "name": "Virtual Person",
  "version": "0.1.0",
  "hostMinStage": "H1",
  "requiresFeature": "virtual_person",
  "description": "Character-01 VRM + props for SamNPlayer",
  "entry": {
    "kind": "asset_pack",
    "webRoot": "web/vrm-viewer",
    "assetsRoot": "assets/character-01",
    "vrmRelative": "assets/character-01/vrm/character-01.vrm"
  }
}
```

| Field | Rule |
|-------|------|
| `apiVersion` | Host accepts `1` only (`pluginhost.CurrentAPIVersion`) |
| `id` | Must be `virtual_person` for the first slot |
| `entry.kind` | `asset_pack` (H1) — web/assets only; Go control plane stays in Animation Studio / later compile-time link |
| Folder name | Prefer `virtual_person/` under Plugins |

### Example drop-in tree

```text
%APPDATA%\SamNPlayer\plugins\
  virtual_person\
    samn-plugin.json
    web\vrm-viewer\          ← stage UI (from tracken)
    assets\character-01\     ← VRM + props (from tracken)
```

Build the pack from Animation Studio with `scripts/pack-samn-plugin.sh` (tracken).

---

## Ownership split (coordinate)

| Side | Provides |
|------|----------|
| **SamNPlayer host** | Plugins folder discovery, `samn-plugin.json` handshake, Settings Enable/Disable/Install, license gate, playback clock (`Tick` / `NowMs`), in-process `virtualperson` core, later overlay slot |
| **tracken (Animation Studio)** | VRM/web assets, `samn-plugin.json`, pack script — **do not** redefine a second host API or duplicate `pluginhost` |

Avoid conflicting APIs: tracken's `virtualperson.Host` is the *plugin-side* view of the same contract (`NowMs`, `EmitAnimation`, later `Device`). SamNPlayer owns `pluginhost.Host` / `Slot`.

---

## Stages

| Stage | What |
|-------|------|
| **H0** | Contract + license gate + Enable stub ([#273](https://github.com/funfunpayer/SamNPlayer/pull/273)) |
| **H1** (this) | Drop-folder discover / Install pack / bind + OnFrame Tick + `virtualperson/` core ([#274](https://github.com/funfunpayer/SamNPlayer/pull/274) / [#275](https://github.com/funfunpayer/SamNPlayer/pull/275)) |
| **H2** (next) | Load pack web overlay; optional ToyHub ownership flip |

---

## H1 acceptance (this slice)

1. Documented host interface (this file).  
2. `pluginhost.Slot` — Enable / Disable / Tick (no-op when disabled) + pack bind.  
3. License feature stamped + `HasFeature` / `EffectiveHasFeature` evaluated.  
4. Settings → drop-folder Install / Open Plugins / Enable/Disable + Give dildo / Start titjob.  
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
  Plugins folder ──discover──► pluginhost.Slot.BindPack
  player.Player ──OnFrame──► App.tickVirtualPersonHost
                               ├─ pluginhost.Slot.Tick (clock / count)
                               └─ virtualperson.Plugin.Tick (bus / activity)
  device.Device ──bridge─► ToyHub (sync OFF — follow-up)
  frontend        ◄─events─ virtualperson:pose (overlay later)
```

## Host APIs (H1 surface)

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

Wails App methods: `PluginsDir`, `ListInstalledPlugins`, `InstallVirtualPersonPack`,
`OpenPluginsFolder`, `VirtualPersonHostStatus`, `EnableVirtualPersonHost`,
`DisableVirtualPersonHost`, `RefreshVirtualPersonPack`, `VirtualPersonGiveDildo`,
`VirtualPersonStartTitjob`.

---

## References

- `license/` — `FeatureVirtualPerson`, `DefaultFeatures`, `EffectiveHasFeature`  
- `docs/LICENSE_SYSTEM.md` — product rules + enforcement  
- `docs/PLUGIN_INSTALL_DE.md` — German end-user steps  
- Animation Studio pack: tracken `samn-plugin.json` + `docs/samnplayer-plugin.md`  
