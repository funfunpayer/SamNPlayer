# Community Edition: seven-day trial of a newly introduced feature

**Scope:** A single optional, time-limited feature. This is NOT a timer on a script and not the existing playback sleep timer. The clock starts only after explicit activation and lasts 7 × 24 hours. User media and output never expire.

## Implementation delivered in this PR

`featurepreview/preview.go` contains a small dependency-free Go domain policy: activate, check remaining time, reject expiry and obvious clock rollback, and isolate feature identifiers. `preview_test.go` covers duration boundaries, no implicit activation, expired retries, isolation, time zones and corrupt state. This code has not yet been integrated with Wails or persisted, and the tests have not yet been run in native Windows CI.

## Remaining tasks (do not pretend they are implemented)

1. Select a feature that is actually implemented and suitable for a temporary Community preview. Register stable feature ID, preview version and display copy.
2. Store state atomically in existing application settings, never inside user scripts. Key per feature, never reset activation on reinstall/upgrade without an explicit decision.
3. Wire an explicit `Start 7-day preview` UI with one-time confirmation and accurate local deadline display. Show remaining time and a clear Preview label.
4. Gate every Wails/Go entry point for the premium feature, not CSS alone. Do not gate script viewing, playback, import/export, safety controls, emergency stop or standard classical generation.
5. After expiry, the preview feature is disabled but existing `.samn`/`.funscript` files, projects, edits and videos remain readable/exportable; where content uses future-only features, provide non-destructive downgrade, warning or Studio edit path.
6. Decide offline/licensing policy. Current local LastSeenAt check only catches basic clock rollback and is not tamper-proof; do not claim it is a security boundary against determined users.
7. Run Go unit tests, cross-edition compile checks, native Wails lifecycle/UI tests, clock boundary/time-zone/offline/error simulations, data non-destruction tests.
8. Document start/end conditions in the Community website, EULA/feature-preview terms as appropriate, changelog and update messaging before public activation.

## AI split and edition contract

Current Free/Community: strong classic CSRT script generation, Play/Edit/Training and safety; later reduce AI scope and remove permanent batch as separately approved by Owner. Keep practical optional AI ROI assistance only if retained in the signed-off Free feature manifest. Internal **Suite** has current training/AI-draft/benchmarks for development. **Next Level/Studio** receives deep local models, continuous curated learning (opt-in), adaptive multi-backend method selection, AI script draft and repair, production batch and full multi-axis Editor.

Existing `cmd/gui-wails/app_edition.go` has `requireStudio()` and frontend `data-studio` as of current main; pending #513 renames it to Suite. The 7-day preview should be a new orthogonal, per-feature right, not a global `Studio()==true` or `Suite()==true` mode. Reuse the API-gate pattern but never grant Suite developer training access as a Community preview by accident.

**Rollout:** finish current Free release first; no user-facing 7-day preview before full persistence and acceptance tests. Future Community releases may temporarily activate selected new premium features with explicit expiry.
