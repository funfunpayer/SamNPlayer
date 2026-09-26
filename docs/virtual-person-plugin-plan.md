# Virtual Person plugin — plan (foundation on #264)

**Status:** foundation scaffold on branch `cursor/virtual-person-plugin-83d0`  
**Primary metaphor:** virtual toy **props** the character uses in-scene (not real-device commands as the main UX).  
**Parallel:** optional Sam Neo 2 / Intiface sync via `ToyHub` (default **off**).

---

## MVP scene

```
give_toy(dildo) → activity(titjob_dildo) → MotionBus owns channels → PoseSample + PropSnapshot
```

Animation MVP: **2D sprite puppet** + prop sprites (`props/dildo.png`). Live2D / glTF later when assets exist.

---

## Package layout (`virtualperson/`)

| File | Role |
|------|------|
| `plugin.go` | Entry: `Start` / `Tick` / `Stop`, GiveDildo, StartTitjob |
| `props.go` | Prop inventory, sockets, Give/Take/Attach |
| `activity.go` | Catalog + scene graph (`titjob_dildo` + stubs with stroke ranges) |
| `bus.go` | MotionBus — activity priority **above** funscript while active |
| `channels.go` | Channel names, mapper, passthrough control |
| `animation.go` | PoseSample for GUI |
| `character.go` | Registry, character-01 stub |
| `chat.go` | Tags `[prop: give=…]` / `[activity: …]`; Echo + **LocalOpenAI** backend |
| `toys.go` | ToyHub — real-device fan-out, sync default off |
| `device_bridge.go` | Thin adapter over `device.Device` |

Host contract: [`docs/PLUGIN_SYSTEM.md`](PLUGIN_SYSTEM.md).

---

## Decisions (this PR)

1. **Assets** — still blocked: copy `character-01/primary.jpg`, create `props/dildo.png` + titjob frames (Project store / Animation Studio). Code already references `SpriteRel` / `RefImage`.
2. **Chat** — **local-first**: `LocalOpenAIBackend` targets OpenAI-compatible localhost (Ollama / LM Studio / Colibri). Falls back to `EchoBackend` offline. No cloud by default.
3. **Real-device sync** — `ToyHub.SetSync(false)` at construction; must be user-enabled.

---

## Next open points

| # | Item | Owner |
|---|------|--------|
| 1 | Asset pack (primary + dildo + frames) | Owner / tracken |
| 2 | GUI overlay wiring (Wails panel + EmitAnimation consumer) | follow-up PR |
| 3 | PluginHost registration inside player tick (ownership rule) | follow-up |
| 4 | Measure: activity vs funscript bus under race | tests already cover priority |

---

## Test

```bash
go test ./virtualperson/
```

Scene-graph, prop attach, activity priority, tag parse, device bridge from mock.
