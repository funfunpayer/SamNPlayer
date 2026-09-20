# Windows OpenCV CSRT (G0.1 → v0.5.13)

**Goal:** Windows portable Generate uses **Go CSRT** (`trackcv`) with no
Python — same quality path as Linux OpenCV builds.

**Status:** build tags unlocked (`cgo && opencv` on all OS). Release still
cross-compiles Windows **without** OpenCV (stub) until this plan ships.
Linux release already links `-tags opencv`.

Related: `docs/PRODUCTION_ROADMAP.md` G0.1, issue #120, `docs/ENGINE.md`.

---

## Why not MinGW cross-compile from Ubuntu?

OpenCV+contrib (CSRT lives in `opencv_tracking`) for `x86_64-w64-mingw32`
is painful to package on the Linux release runner. GoCV’s proven path is
**build on `windows-latest`** with MinGW OpenCV (cached), then ship DLLs
beside the `.exe`.

We keep our slim `trackcv` wrapper (not full GoCV) — only link what
`cv.cpp` needs.

---

## Approach (chosen)

1. **CI / Release job on `windows-2022`**
2. Install **MSYS2** MinGW64 + `mingw-w64-x86_64-opencv` (OpenCV **5.x**,
   pkg-config name `opencv5`, includes `libopencv_tracking` / CSRT)
   *or* cache a MinGW OpenCV 4 build (GoCV-style) if 5.x API breaks us.
3. `CGO_ENABLED=1` + `-tags opencv` + `CGO_CPPFLAGS` / `CGO_LDFLAGS` /
   PATH to MinGW `bin` (DLLs).
4. Build:
   - `go build -tags opencv ./cmd/cli` → `SamNPlayer-cli-windows-amd64.exe`
   - Wails GUI with same tags / env
5. Portable zip: copy required `libopencv_*.dll` (+ deps) next to
   `SamNPlayer.exe` (same pattern as ffmpeg).

Linux job stays as today (`pkg-config opencv4` on ubuntu).

---

## Build tags (done in tree)

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

## Out of scope here

- AI Train / ultralytics (still Python)
- Research backends in GUI
- License enforcement
- Heuristics package (G1)

---

## Implementation checklist

- [x] Remove `!windows` from OpenCV build tags
- [ ] `windows-opencv` CI job (MSYS2 OpenCV + `go test -tags opencv`)
- [ ] Release: Windows artifacts from `windows-latest`, not MinGW cross
- [ ] DLL list + portable zip copy
- [ ] Owner clip gate on real Windows PC
- [ ] Tag **v0.5.13**
