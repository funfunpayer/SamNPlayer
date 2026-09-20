# Windows OpenCV CSRT (G0.1 → v0.5.13)

**Goal:** Windows portable Generate uses **Go CSRT** (`trackcv`) with no
Python — same quality path as Linux OpenCV builds.

**Status:** build tags unlocked; CI `Go (OpenCV Windows)` required green;
release builds Windows on `windows-latest` + MSYS2 and ships OpenCV DLLs
in the portable zip (`scripts/collect-mingw-opencv-dlls.sh`).

Related: `docs/PRODUCTION_ROADMAP.md` G0.1, issue #120, `docs/ENGINE.md`.

---

## Why not MinGW cross-compile from Ubuntu?

OpenCV+contrib (CSRT in `opencv_tracking`) for `x86_64-w64-mingw32` is
painful on the Linux release runner. Proven path: **build on
`windows-latest`** with MSYS2 MinGW OpenCV, then ship DLLs beside the
`.exe`.

We keep the slim `trackcv` wrapper (not full GoCV).

---

## Approach (shipped in workflows)

1. CI / Release job on **`windows-latest`**
2. **MSYS2** MinGW64 + `mingw-w64-x86_64-opencv` (OpenCV **5.x**,
   pkg-config `opencv5`, includes tracking/CSRT)
3. `CGO_ENABLED=1` + `-tags opencv` + `PKG_CONFIG_PATH` + MinGW `gcc/g++`
4. Build CLI + Wails GUI with `-tags opencv`
5. Portable zip: `ffmpeg` + OpenCV/MinGW DLLs via
   `scripts/collect-mingw-opencv-dlls.sh`
   (must run under **MSYS2 MINGW64**, not Git Bash — `/mingw64` must be
   mounted)

Linux stays on Ubuntu `pkg-config opencv4`.

---

## Build tags

| File | Constraint |
|------|------------|
| `trackcv` real impl + `cv.cpp` | `cgo && opencv` |
| `trackcv` stub / `native_track_stub` | `!opencv \|\| !cgo` |
| Linux cgo | `#cgo !windows pkg-config: opencv4` |
| Windows cgo | `#cgo windows pkg-config: opencv5` (+ CXXFLAGS) |

Default clone (no `-tags opencv`) still builds everywhere.

---

## Verify on a Windows box (owner)

After a tagged build with OpenCV DLLs:

1. Fresh folder, unzip portable — **no** Python installed.
2. Generate → Log: `Path: Go CSRT` / `trackcv`.
3. Same clip ROI as Linux; Quality Doctor not worse than Python CSRT.

---

## Out of scope

- AI Train / ultralytics (still Python)
- Research backends in GUI
- License enforcement
- Heuristics package (G1)

---

## Implementation checklist

- [x] Remove `!windows` from OpenCV build tags
- [x] `windows-opencv` CI job (MSYS2 OpenCV + Go CSRT tests)
- [x] Release: Windows artifacts from `windows-latest` + DLL zip
- [x] DLL collector script
- [ ] Owner clip gate on real Windows PC
- [ ] Tag **v0.5.13**
