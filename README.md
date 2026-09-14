# SamNPlayer

Controls a SVAKOM Sam Neo 2 / Sam Neo 2 Pro over BLE using a `.funscript`
file. Available as a CLI (`cmd/cli`) or a native desktop GUI
(`cmd/gui-wails`, built with [Wails](https://wails.io)): a Go backend and a
lightweight vanilla JavaScript frontend using the operating system's web
engine. Windows 11 includes WebView2; Linux uses WebKitGTK and macOS uses
WKWebView. No Electron or bundled browser engine.

## Ready-to-run binaries

Windows and Linux binaries are attached to each
[release](https://github.com/funfunpayer/SamNPlayer/releases), together with
`checksums.txt`. The GUI executable is approximately 13 MB, considerably
smaller than typical Electron applications because it does not include its
own browser engine. Download and double-click; no installation is required.

Binaries are deliberately **not** tracked in the repository (see
`.gitignore`): they change completely with each build and are generated
by the release workflow.

**Bluetooth adapters on Windows 11:** the Buttplug/Intiface developers
(the community behind the device protocol reference) recommend the
**TP-Link UB500** or **Asus USB-BT500**. Their guidance explicitly advises
against built-in motherboard/laptop Bluetooth radios without an external
antenna on Windows/Linux.
[Source: Intiface Bluetooth hardware guidance](https://intiface.com/docs/intiface-central/hardware/bluetooth/).

## Video playback with real synchronization

The playback tab embeds a real `<video>` element, without a separate window
or VLC process. It automatically looks for a video with the same basename
next to the selected Funscript (`szene.mp4` + `szene.funscript`, the same
convention used by MultiFunPlayer/ScriptPlayer). When the option to follow
the actual video position is enabled, the video's playback position drives
the device output (`player.Sync()` in `player/sync.go`), rather than an
independent clock. Pausing, seeking, and changing the video's playback speed
therefore affect the device directly. Extended-O can optionally pause the
video for its hold duration as well.

Implementation note: WebView2/WebKitGTK reject direct `file://` URLs in the
`<video>` tag for cross-origin reasons; this was tested, not assumed. When
loading the first video, the app starts a small local HTTP server bound to
`127.0.0.1`. It serves only the selected file and supports range requests for
seeking. The video is embedded through that server.

## Building from source

Requires Node.js/npm for frontend bundling in addition to Go.

```bash
go build ./cmd/cli                              # CLI
cd cmd/gui-wails && wails build -tags webkit2_41 # GUI (Linux/macOS)
```

The `webkit2_41` tag is required on Ubuntu 24.04+, which provides WebKitGTK
4.1 rather than 4.0. On older systems, build without this tag.

Cross-compiling for Windows from Linux/macOS (the method used for the
supplied executable): the Windows target itself is pure Go and does not
need a C compiler; local Linux GUI dependencies do.

```bash
cd cmd/gui-wails
GOOS=windows GOARCH=amd64 wails build -platform windows/amd64 \
  -ldflags "-X github.com/funfunpayer/SamNPlayer/update.Version=v0.1.0"
```

## Releases and automatic updates

For each `vX.Y.Z` tag, `.github/workflows/release.yml` builds Windows and
Linux binaries for both GUI and CLI and publishes a GitHub release with
`checksums.txt`. The app (`update/update.go`) can check for updates at
startup, configured in the settings tab. With the user's consent it can
download the update and restart, verifying SHA256 against the checksum
file and checking that the download URL actually comes from github.com.

The release repository is already configured as `funfunpayer/SamNPlayer`.
[v0.2.1](https://github.com/funfunpayer/SamNPlayer/releases/tag/v0.2.1)
was successfully built for Windows and Linux on September 14, 2026.
For future releases, follow [CONTRIBUTING.md](CONTRIBUTING.md#versions).

## Generating scripts from video

In the script-generation tab, choose a video, drag a region over the target
in the first frame using the `<canvas>` preview, and generate the script.
The generator uses classical computer vision: OpenCV CSRT tracking,
Savitzky–Golay smoothing, and peak detection
(`generator/generate_funscript.py`). It uses **no trained AI model**.
The script is embedded in the Go binary using `go:embed`; Python and its
packages are the external runtime requirements.

**Requires Python 3.9+** and the packages in `generator/requirements.txt`.
Run `pip install -r generator/requirements.txt`, or use the dependency-check
button to find out what is missing.

For stronger generation capabilities such as AI object detection instead
of manual ROIs, VR support, automatic scene detection, and processing entire
folders, the existing project documentation recommends
[FunGen 2](https://fungen.app), which runs locally and is free for personal
use. FunGen is **not open source** (PolyForm Strict License), so it is not
integrated here. Its output is a normal `.funscript` file that this player
can play directly.

## Logging

Structured logging (`logging/`, `log/slog`) writes to
`<user configuration directory>/SamNPlayer/logs/SamNPlayer.log`
(on Windows: `%AppData%\SamNPlayer\logs\`), with rotation at 5 MB.
The log level (debug/info/warn/error) can be changed in the settings tab;
changes take effect immediately and persist across restarts.

## Persistent settings

A small JSON store (`settings.json` in the same directory as the log file)
provides persistence; Wails does not have a built-in preferences API like
some other GUI toolkits. The following settings survive restarts:

- Automatic update checks at startup.
- Log level.
- All playback defaults: mock mode, sync mode, tick, maximum speed, and Extended-O.

## Community-inspired improvements

Added after reviewing MultiFunPlayer, a major community player:

- **Script heatmap:** a color-coded intensity bar covering the entire script,
  directly below the video preview (blue = calm, red = intense).
- **Soft start:** ramps up over a configurable interval (500 ms by default)
  instead of immediately jumping to the full script value.
- **Playback smoothing:** adjustable in the playback tab as well as the
  generator; 0 disables it.
- **Keyboard shortcuts:** Space starts/stops playback; E triggers Extended-O.
- **Vibration-only and suction-only sync modes** (`funscript/mapper.go`):
  route imported scripts to a specific channel instead of always using both.

Both vibration and suction on the Sam Neo 2 accept scalar control, as
checked against the Buttplug Rust source: `OutputCommand::Constrict` uses
the same generic scalar value type as `OutputCommand::Vibrate`, with 6
rather than 11 levels. An earlier assumption that suction was limited to
five fixed rhythm patterns was incorrect. It came from a forum report about
the official SVAKOM app's preset UI, not the raw protocol. The resulting
artificial suction delay was removed; suction now follows the same control
approach as vibration. Actual firmware resolution still requires hardware
validation, as described in `HANDOFF.md`.

## Training mode

A separate tab provides standalone ramp-up/ramp-down cycles, independent of
video or script (`player/training.go`). The two techniques come from
clinical/community descriptions rather than being invented for the app:

- **Stop-start:** ramp up, hold briefly, drop to approximately zero, pause,
  and repeat; the classic method described by Semans in the 1950s.
- **Plateau/edging:** remain at a high level after ramping up instead of
  dropping all the way down.

Both support configurable cycle counts and timings, optional increases
between cycles, and a choice of vibration, suction, or both channels.
Each session is logged to
`<log directory>/sessions/training-<technique>-<timestamp>.jsonl`, with one
JSON object per cycle containing timestamp, channel, peak value, hold time,
and duration. These logs support later tuning.

## Marked range and automatic Extended-O

Drag on the playback heatmap to mark a time range, such as a section near
the end. With automatic Extended-O for the marked range enabled, Extended-O
triggers once per playback when the video-synchronized or clock-driven
position reaches that range. The marker is stored in a small JSON file
beside the script (`szene.funscript.marker.json`), rather than modifying the
Funscript and affecting other players. It can be removed with the
clear-marker control.

## Stability fixes

- **No silent overwrites in the generator:** if a `.funscript` already exists
  beside the video, whether handmade, downloaded, or generated earlier, the
  generator asks before replacing it. Testing previously caused exactly
  this kind of irreversible data loss.
- **Keyboard focus after file dialogs:** native file dialogs could leave
  focus on a button, making Space/E shortcuts appear unresponsive. Focus is
  now explicitly returned to the window.
- **Playback and training are mutually exclusive:** both control the same
  physical device. Starting both could previously overwrite the cancellation
  function and make the first session impossible to stop. The second session
  now receives a clear error. An interactive test started training with a
  long hold, attempted playback, confirmed rejection, and then verified that
  a new session could start immediately after stopping.
- **GUI versus CLI update selection:** `AssetForThisPlatform()` previously
  distinguished only OS/architecture, although both Windows binaries end in
  `-windows-amd64.exe`. Depending on server response order, the GUI could
  replace itself with the CLI. Selection now explicitly distinguishes
  `gui` and `cli`.

## Automatic region selection

The generator can find a motion region automatically
(`generator/auto_roi.py`, the automatic-region button) instead of requiring
a manually marked region. The method follows sections 5–6 of the project
specification:

1. Dense Farneback optical flow over a grid of image cells.
2. **Camera-motion compensation:** feature tracking and an affine RANSAC
   model estimate global motion, which is subtracted from the flow field.
   Without this, a camera pan would look like motion everywhere and region
   selection would become arbitrary.
3. Rank motion by **periodicity**, not magnitude. FFT measures how strongly
   one frequency dominates within 0.1–4 Hz. Rhythmic motion outranks random
   movement.
4. Merge the best cells into a connected region.

This deliberately uses **no trained detection model**: it looks for rhythm,
not particular objects. It is content-independent, needs no model weights,
and can work with material for which no suitable model exists. Tests with
static objects, random motion, and camera pans found the rhythmic region
in each case.

The result is a suggestion: the region can still be adjusted manually in
the preview.

## Known limitations

- **Sam Neo 2 protocol:** checked against the official Buttplug Rust source
  (see comments in `device/protocol.go`), but not tested on real hardware.
- **Video codecs:** `<video>` supports the codecs available in the webview
  engine, such as H.264/MP4, WebM/VP9, and AV1. Older or unusual codecs such
  as MPEG-4 Part 2 (`mp4v`) and Xvid are not supported; re-encode with
  `ffmpeg -c:v libx264` if necessary.
