# virtualperson

In-process Virtual Person foundation for SamNPlayer.

**Primary:** virtual toy **props** + activities (MVP: give dildo → titjob).  
**Parallel:** optional real Sam Neo 2 / Intiface sync via `ToyHub`.

```go
p := virtualperson.NewPlugin(host)
_ = p.GiveDildo()
_ = p.StartTitjob(0.5, 30)
_ = p.Start(ctx)
_ = p.Tick(scriptPos, vibe, suck) // activity drives pose + prop snapshot
p.Toys().SetSync(true)            // optional real-device output
```

See Project Store plan: `docs/virtual-person-plugin-plan.md`.

Host contract for SamNPlayer: [`docs/PLUGIN_SYSTEM.md`](../docs/PLUGIN_SYSTEM.md).
Primary Animation Studio product code: `funfunpayer/tracken`.
