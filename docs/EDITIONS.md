# Editions — SamNPlayer Free, SamNPlayer Suite, SamNPlayer Next

## Product lines (Owner, 12 Oct 2026)

| Name | What it is | In the code |
|---|---|---|
| **SamNPlayer Free** | The software for customers. | Default build (no tag), edition name `free` |
| **SamNPlayer Suite** | **Our internal extended version** with AI training and the batch test system. It keeps the name "Suite". | `-tags suite`, edition name `suite` |
| **SamNPlayer Next Level 2.0** | Future donation-supported freeware, developed as a separate software generation using proven parts of SamNPlayer. **No paid unlock or licence-key plan.** | Separate public [project and roadmap](https://github.com/funfunpayer/SamNPlayer-Next-Level); not yet released |

Rules:
- **Free stays free.** Free is also our test system.
- **Which features later move out of Free** (for example batch processing) is the Owner's decision alone.
- **Safety stays in every edition:** emergency stop and device limits.
- **Both Free and Next Level 2.0 are free and supported by voluntary donations.** There is no paid Studio product, €40 unlock, compulsory subscription or timed trial planned. The historical paid-Studio proposal in PR #510 is superseded.
- The previously proposed seven-day feature preview was cancelled; PR #515 was closed without merge.
- Later Free scope reductions (smaller AI surface and removal of permanent batch) require their own reviewed change, migration documentation and regression tests; they are **not part of the current Free release**.

## Free vs Suite today

Owner decision 11 Oct 2026: AI training and learning belong to our internal system, not to the build customers get. One code base, one build switch.

| | Free (default) | Suite (`-tags suite`) |
|---|---|---|
| Who | customers (release downloads) | our team (release files `SamNPlayer-suite-…`, private) |
| Play, Create, Edit, Training (device sessions), Tools | yes | yes |
| AI **use**: AI tip/region detection, scene proposals, contact points, Apply AI setup, AI server | yes | yes |
| AI Train tab (labelling, dataset, YOLO/ONNX training, profile model) | — | yes |
| Golden-clip benchmark tab | — | yes |
| Scene-map learning (collect, export, delete, "Suggest ignores from learning") | — | yes |
| Teacher contact candidates review (dataset labels) | — | yes |
| "Remember scene + style" (profile-model labels) | — | yes |
| AI draft / imitation export | — | yes |
| Settings → "Learn from judgments" (quality model) | — | yes |

## How it works

- **Package `edition`:** `Suite()` is a compile-time constant. `suite_on.go` carries `//go:build suite`; `suite_off.go` is used otherwise. `Name()` is `"suite"` or `"free"`.
- **GUI bindings:** the Suite-only bindings exist in both builds, so the Wails JS API is the same. In the Free build they return `errSuiteOnly` (`cmd/gui-wails/app_edition.go`, `requireSuite()`).
- **Frontend:**
  - Suite-only controls carry `data-suite`.
  - `main.js` sets `html[data-edition]` to `free` or `suite` from `IsSuiteBuild()`.
  - A CSS rule hides Suite-only controls in Free.
- **Releases (`release.yml`)** build both editions:

  | Edition | Release files |
  |---|---|
  | Free | `SamNPlayer-gui-…`, `SamNPlayer-cli-…`, `SamNPlayer-portable-…` |
  | Suite | `SamNPlayer-suite-gui-…`, `SamNPlayer-suite-cli-…`, `SamNPlayer-suite-portable-…` |

  `scripts/publish-public-release.sh` publishes only the Free binaries, so Suite files stay private.
- **In-app updater:** it matches the exact file name for its own edition (`guiAssetKind()` returns `gui` or `suite-gui`). Free never updates itself to Suite, and Suite never updates itself to Free.

## Building the Suite edition

```bash
# GUI
cd cmd/gui-wails && wails build -tags "suite opencv"      # + webkit2_41 on Linux
# CLI
go build -tags "suite opencv" ./cmd/cli
# tests
go test -tags suite ./edition/ ./cmd/gui-wails/
```

## Not yet split

- The generator's embedded Python files (`generator/*.py`, including the training scripts) are still embedded in both builds.
- The Free build never offers or runs the training paths.
- Moving the training scripts into a Suite-only embed is a follow-up.

## History

- Until 12 Oct 2026 the internal build was called "studio" (`-tags studio`), and the default build was called "customer".
- No release was tagged under those names, so no installed build looks for `studio-` files.
