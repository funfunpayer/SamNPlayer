# Training verification — 22 September 2026

Verified baseline: `08f7235a225650770b668e10bd2c036b2c2bfd41` on main. This is a verification record, not a fix or hardware acceptance.

## Results

| Check | Result | Evidence |
|---|---|---|
| Player engine | PASS | `go test -race ./player -count=1`, 11.216 s; includes carried-channel interruption, feedback StartLevel scaling, level mirror |
| Existing CI | PASS | [Tests run](https://github.com/funfunpayer/SamNPlayer/actions/runs/35733582374), including Go, frontend, Python and OpenCV jobs |
| Production frontend | FAIL: missing clip assets | `npm ci && npm run build` exits successfully, but Vite leaves directory-based `new URL` unresolved; 16 source PNG frames, zero clip PNGs in dist |
| Pending/late levels after done | FAIL | Actual JS handlers executed in Node VM with mocked DOM and controlled RAF queue restore nonzero values after done |
| Completion before start response | FAIL | A mocked StartTraining emits done before resolving; start() subsequently marks session running |

## Reproduction and acceptance criteria

1. **Clip packaging:** build `cmd/gui-wails/frontend`, inspect `dist/assets` for all f00–f15 assets and the built URLs. Current `pixel_figure.js` constructs a directory URL, then appends filenames dynamically. A successful Vite exit does not prove asset availability. Fix acceptance: all 16 referenced frames resolve in a production build.
2. **Pending frame after completion:** start a session; emit levels `{vibration:0.8,suction:0.6}`; before executing its requestAnimationFrame callback, emit `training:done`; flush the callback. Observed display: 0.8/0.6 with running=false. A fresh late levels event after done also restores values. Fix acceptance: pending and late updates cannot revive an ended session's display; new sessions still receive initial updates.
3. **Start response race:** make StartTraining emit `training:done` before its promise resolves. Observed: start() subsequently calls setRunningState(true). Fix acceptance: the completed session stays stopped and start failures restore idle controls.

Lifecycle probes execute the real setRunningState, applyLiveLevels, start and event-handler source with dependency stubs; they are deterministic unit-level reproductions, not full browser tests. Chromium launch was blocked by environment socket permissions. No physical device was tested. No production code was changed.

## Handoff

[Cursor/Claude notified on #177](https://github.com/funfunpayer/SamNPlayer/pull/177#issuecomment-5777814009). The integrated engine fixes must not be re-landed from the old codex branch. The three frontend findings remain open; no agent has confirmed taking their implementation in this handoff.

The newer lane proposal [#184](https://github.com/funfunpayer/SamNPlayer/pull/184) assigns ChatGPT **MT-Go verification** of #183 (`94b2bae`): coast, reacquisition, tracking-loss UI, and partner-loss evidence on 1–2 available real clips. Report pass/fail or blocked for the MT-ID gate; synthetic tests alone cannot establish real-clip identity preservation. Then produce **MT-Speed** notes / optional small scripts-only spike. MT-Seed stays with Cursor, MT-Debug with Claude. Do not alter Generate defaults or reimplement coast/reacquisition. This report does not claim the newer MT-Go commit has been tested.
