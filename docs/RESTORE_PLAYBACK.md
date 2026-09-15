# Restore playback.js on this branch

The GitHub file API truncated `cmd/gui-wails/frontend/src/playback.js`.
Apply this locally on `feat/polarity-roi2-o-zone` (does not touch main):

```bash
git checkout feat/polarity-roi2-o-zone
git checkout main -- cmd/gui-wails/frontend/src/playback.js
git apply docs/playback_hooks.patch
git add cmd/gui-wails/frontend/src/playback.js
git commit -m "fix(playback): restore main player + O-key / polarity / ring-down hooks"
git push origin feat/polarity-roi2-o-zone
```

Patch is against current `main` playback.js.
