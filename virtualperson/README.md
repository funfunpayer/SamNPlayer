# virtualperson

In-process Virtual Person core for SamNPlayer (H1).

## What lands here

- Prop inventory (`dildo`, sockets, Give/Take/Attach)
- Activity catalog + scene graph (`titjob_dildo` + stroke ranges)
- MotionBus (activity priority above funscript while active)
- Passthrough control + channel mapper + animation driver
- Chat tag stubs + local-first backend (no GUI yet)
- ToyHub (sync **default off** — device ownership flip is a follow-up)

## Host wiring

See `docs/PLUGIN_SYSTEM.md`. The Wails app enables the host in Settings;
playback `OnFrame` calls `Plugin.Tick` when running. Pose samples are emitted
as `virtualperson:pose` events — sprite overlay UI is not in this slice.

## Tests

```bash
go test ./virtualperson/
```

## Source

Cherry-picked from parked draft [#264](https://github.com/funfunpayer/SamNPlayer/pull/264)
— do not force-merge that branch wholesale.
