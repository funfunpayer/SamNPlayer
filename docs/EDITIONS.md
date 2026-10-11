# Editions — customer build and studio build

Owner decision 11 Oct 2026: AI training and learning belong to our internal
system, not to the build customers get. One code base, one build switch.

| | Customer build (default) | Studio build (`-tags studio`) |
|---|---|---|
| Who | everyone (release downloads) | internal team |
| Play, Create, Device, Training (device sessions) | yes | yes |
| AI **use**: AI tip/region detection, scene proposals, contact points, Apply AI setup, AI server | yes | yes |
| AI Train tab (labelling, dataset, YOLO/ONNX training, profile model) | — | yes |
| Golden-clip benchmark tab | — | yes |
| Scene-map learning (collect, export, delete, "Suggest ignores from learning") | — | yes |
| Teacher contact candidates review (dataset labels) | — | yes |
| "Remember scene + style" (profile-model labels) | — | yes |
| AI draft / imitation export | — | yes |
| Settings → "Learn from judgments" (quality model) | — | yes |

## How it works

- Package `edition`: `Studio()` is a compile-time constant
  (`studio_on.go` with `//go:build studio`, `studio_off.go` otherwise).
- GUI bindings for studio tools stay in both builds (the Wails JS API is the
  same) but return `errStudioOnly` in the customer build
  (`cmd/gui-wails/app_edition.go`, `requireStudio()`).
- Frontend: studio-only controls carry `data-studio`; `main.js` sets
  `html[data-edition]` from `IsStudioBuild()` and a CSS rule hides them in
  the customer build.
- Releases (`release.yml`) build without the tag = customer build.

## Building the studio edition

```bash
# GUI
cd cmd/gui-wails && wails build -tags "studio opencv"      # + webkit2_41 on Linux
# CLI
go build -tags "studio opencv" ./cmd/cli
# tests
go test -tags studio ./edition/ ./cmd/gui-wails/
```

## Studio-only embed

Training Python scripts (`train_yolo_model.py`, `bootstrap_yolo_dataset.py`,
`export_yolo_onnx.py`) and `requirements-ai-train.txt` live under
`generator/studio/` and are embedded only in the studio build
(`//go:build studio` in `studio_embed.go`). The customer build never ships
them. `writeScriptToTemp` copies them into the temp dir only when the studio
tag is set.
