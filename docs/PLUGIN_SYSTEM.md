# Plugin system — drop-folder host contract

**Status:** H1 infra kept — drop-folder discover / Install / Open Plugins.  
**Virtual Person product:** **PARKED** (draft [#264](https://github.com/funfunpayer/SamNPlayer/pull/264); no product UI/host Enable/scene steps on tip).  
**Related:** package [`pluginhost/`](../pluginhost/) · reserved pack id `virtual_person` · license feature `virtual_person` (stamped, unused while parked) · Animation Studio [`funfunpayer/tracken`](https://github.com/funfunpayer/tracken)

---

## End-user load path (infra only)

1. Drop a pack folder into the SamNPlayer **Plugins** folder  
   (`%APPDATA%\SamNPlayer\plugins` on Windows), **or** Settings → Plugins → **Install pack…**
2. Packs must contain `samn-plugin.json` at the root
3. Product Enable / OnFrame host / overlay UI is **not** shipped while Virtual Person is parked

Everyday Create / Play / Train are unchanged.

---

## Goal (when unparked)

Virtual Persons **attach to SamNPlayer** as a plugin / extension — they do not
replace Generate / Play / Train. Full product surface lives in #264 / Animation Studio.

**What this is not**

- Not moving Animation Studio into this repository  
- Not a marketplace  
- Not dynamic Go `.so` loading for end users  
- Not Everyday CSRT changes  

---

## License

| Decision | Choice |
|----------|--------|
| Feature id | **`virtual_person`** (reserved; stamped on standard keys) |
| Monetization | Included in the standard €40 / year key when product returns |
| Enforcement | Remains **off** — do not flip without Owner OK |

---

## Package layout + version handshake

### `samn-plugin.json` (required)

```json
{
  "apiVersion": 1,
  "id": "example_pack",
  "name": "Example Pack",
  "version": "0.1.0",
  "hostMinStage": "H1",
  "requiresFeature": "",
  "description": "Asset pack for SamNPlayer",
  "entry": {
    "kind": "asset_pack",
    "webRoot": "web",
    "assetsRoot": "assets"
  }
}
```

| Field | Rule |
|-------|------|
| `apiVersion` | Host accepts `1` only (`pluginhost.CurrentAPIVersion`) |
| `id` | Any safe id; `virtual_person` reserved for parked product |
| `entry.kind` | `asset_pack` (H1) |

---

## Ownership split

| Side | Owns |
|------|------|
| **SamNPlayer host** | Plugins folder discovery, `samn-plugin.json` handshake, Settings Install / Open folder |
| **Parked product** | In-process scene bus, Enable host, Give/Titjob, pose overlay (#264) |

---

## Stages

| Stage | Scope |
|-------|--------|
| **H1 infra** (this tip) | Drop-folder discover / Install pack / Open Plugins ([#275](https://github.com/funfunpayer/SamNPlayer/pull/275)); product Enable/tick scrubbed for Rel35 |
| **Parked** | `virtualperson/` core, Settings Enable, OnFrame tick, overlay |

Wails App methods (shipped): `PluginsDir`, `ListInstalledPlugins`, `InstallPluginPack`,
`OpenPluginsFolder`.

Packages: `pluginhost/` (manifest + paths + Slot stub). License keeps
`FeatureVirtualPerson` in `DefaultFeatures` for key compatibility while product is parked.
