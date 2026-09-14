# SamNPlayer — project overview

This document deliberately lives **inside the repository**, rather than
in a separate PDF. It travels with the code and should be updated alongside
code changes so the two do not drift apart.

Status: September 2026.

---

## What the project is

A native desktop application (Go + Wails) that controls a SVAKOM Sam Neo 2
over Bluetooth, synchronized with a `.funscript` file. It also includes a
generator that creates `.funscript` files from video using classical
computer vision, without neural networks.

Guiding principles: **do not add what is not needed**, and support every
claim about quality or speed with measurements rather than estimates.

**Stack:** Go 1.25.0 or later (see `go.mod`), Wails v2.15, vanilla JavaScript
(no framework), and Python 3.9+ with OpenCV/scipy/numpy. Python scripts are
embedded with `go:embed` and launched as a subprocess at runtime.

---

## Structure

```text
cmd/gui-wails/     Wails GUI (main product), five tabs
cmd/cli/           CLI version (without generator)
device/            BLE protocol and transport, mock device
player/            Playback, synchronization, training mode
funscript/         Parser, metadata, mapping
generator/         Python pipeline and Go wrapper
motionx/           RDP reduction and motion-state classification (salvaged)
videox/            ffprobe/ffmpeg wrappers (salvaged, not yet connected)
logging/ update/   Logging, automatic updates through GitHub releases
```

---

## What works

**Playback:** load a script, play a synchronized video, display the
Funscript curve with a moving playhead below the video, heatmap, and markers.
Includes **script offset**, saved per script and applied immediately during
playback, because imported scripts rarely match the user's video cut
exactly. Also includes **section looping** and keyboard controls: Space,
←/→ for 5-second seeks or 1 second with Shift, `,`/`.` for fine stepping,
1–9 for jumps, `+`/`−` for offset, `L` for looping, and `E` for Extended-O.

The offset is applied in exactly **one** place: where video time maps to
script time (`ReportVideoPosition`). Applying it again to the frontend
display would make the curve and device drift apart.

**Two device connection options:** direct Bluetooth using the local adapter
without additional software, or WebSocket access to a running
**Buttplug server** (Intiface Central). The latter requires no Bluetooth
adapter on the player computer and works with devices supported by Buttplug.
Intiface handles connection, pairing, and reconnection, which are often the
most error-prone parts. The implementation follows the open protocol
specification without copying source code. `gorilla/websocket` was already
present through Wails, so no new dependency was needed.

The server can run **on another device**, for example Intiface Central on a
phone with the player on a computer. The address field accepts an IP such
as `192.168.1.50`, a host and port such as `192.168.1.50:12345`, or a complete
`ws://` address. Connection type and address are remembered after a
successful connection. Errors distinguish local from network connections
because likely causes differ: a server that has not started versus the
wrong Wi-Fi network, IP address, or a firewall.

Tested against a simulated Buttplug server in seven checks, including a
device without a suction channel, a missing server, address normalization,
and the distinction between local and network errors.

**Direct Bluetooth remains the default.** Intiface is an additional option.

**Device:** a dedicated tab provides connect/disconnect, device name, BLE
address, signal strength, vibration/suction function tests, and a raw-value test.

**Training:** Semans stop-start and a plateau variant; an interrupt control
ends the current cycle rather than the entire session. An arousal scale
from 1–10 adjusts the next cycle. Sessions are logged.

**Generator:** two backends — CSRT tracking with a marked region, or optical
flow without a region. Includes automatic region selection, scene-cut
detection with region selection per scene, camera compensation, adaptive
keyframes, RDP, speed limiting, axis selection, automatic retry, parallel
batch processing, caching, and measurement reports with user feedback.

**Appearance memory:** the tracker remembers what the region looks like
and searches the whole frame after tracking loss or a scene cut instead
of blindly reinitializing at the last position. This reproduces one useful
capability of object detectors using classical methods. On a three-scene
test video with true motion of 90/100/80 px, the measured ranges were
90.5 / **1.5** / **5.0** px without memory and
90.5 / **101.0** / **80.0** px with memory. On the occlusion video, tracking
loss fell from 323 to 148 out of 500 frames. The remaining 148 frames are
those where the object is actually absent; they are reported as unsuccessful
searches rather than hidden. Below the minimum similarity threshold the
tracker deliberately does not reinitialize: a guessed position would look
like a measurement without being one.

**Two-point measurement:** tracks two regions and uses their **distance**
as the signal. Common translation of both points, whether caused by camera
panning or shared object motion, cancels out instead of needing to be
removed afterward. Zoom can still change image-space distance and must not
be treated as inherently canceled. On a test video combining panning,
shared movement, and an oscillating gap, correlation with the true distance
was −0.01 for one tracker and +0.62 for two trackers.

**Motion profiles:** `--profile standard` for stroke motion and
`--profile weich` for soft tissue. The difference is physical: a rigid
stroke is one movement, whereas soft tissue exhibits damped oscillation
after an impulse. That ringing is the *result* of the impulse, not another
stroke. Without a prominence requirement, each oscillation becomes a keyframe.

On a test video with impulses every 800 ms and ringing at 4 Hz, the output
contained 81 keyframes without prominence filtering and 42 with prominence
0.35, matching roughly 40 impulses. Reconstruction error rose from 0.094
to 0.176 because ringing was deliberately excluded. This is why it is a
selectable profile rather than the default. **Clean stroke signals remain
exactly unchanged:** 42 or 80 keyframes, with identical error.

**Motion signature (scene recognition):** eight measurable scene features:
main direction, motion-center location and spread, number of separate
regions, camera instability, stroke symmetry, and rhythm strength. These
allow the app to recognize similarity to a previously named scene and reuse
parameters that worked there.

**Important distinction:** this does not identify *what* the scene depicts.
Naming an action from its content would require semantic recognition and
a trained model with labeled data. The user supplies the name; the program
transfers it only to similar signatures.

Measured distances: two vertical-motion scenes at different speeds were
0.075 apart; vertical versus horizontal was 0.26–0.32, and versus camera
panning 0.28–0.29. A threshold of 0.15 separated these examples. Above the
threshold, no assignment is made: incorrectly transferred parameters are
worse than no assignment and harder to notice later.

Not yet connected: naming through the UI and applying saved parameters to
generation runs. Region counting currently returns the same value for all
test videos, so it contributes nothing to distinguishing them.

**Learned quality assessment:** user ratings in the measurement report
(usable / borderline / unusable) can train logistic regression over seven
metrics. This deliberately uses plain numpy rather than a neural network,
so weights remain readable and decisions can be inspected. A model is
accepted only if it outperforms the fixed rules in leave-one-out
cross-validation, using at least 12 rated runs and at least 4 per class.
It is stored as `qualitaetsmodell.json` in the configuration directory.
Borderline counts as a failure, favoring an extra user check.

**Script analysis:** divides a loaded script into motion states — stationary,
starting, accelerating, steady, decelerating, and stopping — and displays
a description below it. This catches a gap in other checks: a long flat
section is neither noisy nor arrhythmic, so those checks miss inactivity.

**Minimum spacing between actions:** enforced during generation rather
than merely reported. Peaks and valleys are found separately, so minimum
spacing within either list does not enforce spacing between a peak and the
next valley. On noisy signals, up to 45% of actions were less than 100 ms
apart. The point with the smaller deviation from the line joining its
neighbors is removed, preserving extrema. Reconstruction error rose only
from 0.084 to 0.086; clean signals were unaffected.

**Device compatibility:** checks generated files against limits established
in the Funscript community so results remain useful on other hardware and
in other players. These values come from existing projects: 100 ms minimum
action spacing (the Launchcontrol transmission threshold), 900 ms for the
slowest useful full stroke, position range 5–95, and intensity
`500 × |Δpos| / |Δt|` as a speed metric (the funscript-utils definition,
also used in OpenFunscripter, Funscript.io, and XBVR).

**Quality Doctor:** evaluates timestamps, value range, gaps, speed outliers,
rhythm (spectral concentration of the dense, **detrended** signal), tracking
loss, actual motion amplitude, **active time fraction**, and
**reconstruction error**: how faithfully exported actions represent the
measured signal.

---

## Measured performance

All figures below come from actual runs rather than estimates.

| Measurement | Value |
|---|---|
| CSRT tracker | ~100 ms/frame (97.5% of total runtime) |
| Dense optical flow (Farneback) | ~18 ms/frame |
| Video decoding | 0.7% of total runtime |
| Cache hit | 31.7 s → 0.88 s (36× faster) |
| Flow backend versus CSRT | 12 s instead of 51 s per test video |

**Amplitude fidelity**, with 110 px of true object motion and a realistically
textured background:

| Method | Stationary camera | Panning |
|---|---|---|
| CSRT + feature-based compensation | 112.2 px | 112.8 px |
| Flow backend (old vector correction) | 137.8 px | 145.2 px |
| Flow backend (position correction) | 137.8 px | **100.2 px** |

Camera correction in the flow backend now uses features to correct
positions rather than flow vectors. Panning correlation improved from
0.744 to 0.805.

---

## Third-party code policy

For a project intended for commercial distribution, the source project's
license matters as well as whether code was copied. The following table
records the project's existing license assessment.

| Project | License | Project policy |
|---|---|---|
| Funscript Flow | Apache-2.0 | Code reuse permitted with attribution |
| funscript-utils, launchcontrol | Permissive | Metrics and limits adopted |
| Buttplug (protocol knowledge) | BSD-3 | Protocol facts may be used; code reuse requires attribution |
| **FunGen 1** | **PolyForm Strict 1.0.0** | **Do not reuse** in this project |
| FunGen 2 | Closed binary | No source available to review |

The project excludes FunGen source-code reuse and structurally equivalent
ports because of licensing concerns around derivative works and commercial
use in an MIT-licensed product. Public descriptions of behavior may inform
independent designs. This records the development policy, not a fresh legal
assessment of third-party licenses.

## Code salvaged from fse-generator

Two packages from an earlier project were reviewed individually instead
of merged wholesale:

| Package | Contents | Status |
|---|---|---|
| `motionx` | Iterative RDP returning indices, `Dedup`, motion-state classification | **Connected:** classification is used in script analysis |
| `videox` | `ffprobe` wrapper and ffmpeg grayscale reader | Present, **not connected** |

`motionx` uses only the standard library. `videox` requires **ffmpeg and
ffprobe on PATH**, a dependency the current project avoids by decoding
video through Python/OpenCV. Connecting it now would require every user
to install more software without a demonstrated benefit. It becomes useful
if removing Python altogether is the goal; that is an architectural
decision, not simply copying another file.

Explicitly **not** salvaged, based on the original analysis:
`EstimateTranslation` integrates the camera signal rather than the subject
signal with search grid 4 and reaches its limit within a few frames;
`pattern.Periodicity` measures smoothness instead of periodicity (linear
drift scores 1.000); `candidate.Build` therefore systematically favors the
over-smoothed candidate, the same class of error as the zigzag problem.

## Extensibility

Analysis methods are replaceable components (`generator/backends.py`).
Previously, selection was hard-coded inside the pipeline, requiring a
pipeline edit for every new method.

A backend is a function with a fixed contract:

```text
analyze(video_path, roi, options)
  -> (timestamps_ms, positions, frame_size, scene_cuts, stats)
```

Two requirements are essential: positions must use **image coordinates**,
not normalized values, otherwise dynamic-range processing and normalization
would silently be bypassed. Also, `stats["vertical_range"]` must contain
amplitude **in pixels**. Normalization permanently removes that information,
but the Quality Doctor needs it to distinguish real motion from amplified
jitter.

Custom methods are Python files in the plugin directory, registered with
`register(name, func, description)`. `--list-backends` lists available
methods and their origins. Contract violations fail immediately with a
message identifying what is missing; otherwise a plugin returning one row
too few would appear merely to produce an inexplicably poor script.

There is deliberately **no sandbox**: plugins run with the same permissions
as the application.

## Fixed bugs affecting users

**Incomplete Python-module embedding:** four files were embedded but only
two were written to the temporary directory. This was invisible in the
source tree, where all modules sit together. In the packaged executable it
caused **three apparently unrelated failures**: `--backend flow` failed,
the learned quality model was never found, and device checking silently
never ran because its import was wrapped in `try/except`. The entire
Python directory is now embedded with `go:embed *.py`.
`generator_embed_test.go` reads actual imports from the source and verifies
that each module reaches the temporary directory. A manually maintained
list would repeat the original mistake.

**Keepalive maintained only one channel:** it repeated only the last sent
packet. If that packet controlled suction, vibration was not maintained,
and vice versa, particularly during pauses and Extended-O, when keepalive
is needed most. Both channels' states are now stored and repeated.
Unchanged packets are also suppressed: integer levels mean 100 ramp steps
produce at most 11 distinct packets. The rest were ineffective round trips,
potentially up to 40 per second for two channels updated every 50 ms.

## Known limitations

**The flow backend overestimates amplitude even with a stationary camera:**
137.8 px instead of 110. This is not a camera-correction bug, because no
correction applies in that case; it is a property of the center estimator.
After normalization to 0–100 the effect is smaller than the raw numbers
suggest, and the shape is accurate (correlation 0.941).

**All quality thresholds were calibrated on synthetic videos:** clean sine
motions with known ground truth. Real material is less regular and
systematically scores lower. The measurement report exists so thresholds
can be recalibrated using real runs **and** human ratings.

**The generator produces weaker motion than FunGen — partly explained.**
On the same film, median action jump was 6.5 versus 44.5, jumps greater than
50 points were 0.0% versus 34.7%, and time in the 40–60 middle band was 33.3%
versus 14.2%. Our script jittered around the middle instead of alternating
between extremes.

One cause was identified and fixed: global normalization used only 30 out
of 100 points in half of all 6-second windows. Moving-window dynamic-range
processing raises average motion strength from 10.1 to 26.2.

The remaining cause is **not post-processing**, as measurements now show.
On synthetic video with realistically varying speed (1.2–2.2 Hz), the
pipeline reaches intensity 162.1 versus an ideal 161.4, effectively without
loss. Marked-region size makes no difference either: 112, 111, and 110 px
for 40×40, 70×70, and 120×120 regions. Smoothing attenuates realistic stroke
frequencies by only 0–8%.

The remaining explanation is the measured quantity itself. A single region
measures that image area's displacement, while real material often depends
on the **relative** motion between two bodies, which can be much larger
than either one's absolute displacement. Two-point measurement (`--roi2`)
addresses this. Automatic detection of both regions is not yet good enough
according to measurements (see `find_two_rois`).

The original video is needed to settle this; the exported script alone
cannot recover the original signal.

**Not tested on real hardware:** connection, playback, and training have
only been verified against `device.Mock` and simulated protocol behavior.

**Actual device resolution is unknown:** the 0–10 vibration and 0–5 suction
ranges come from Buttplug's device configuration. They describe Buttplug's
quantization, not necessarily firmware limits. The device tab's raw-value
test is intended to resolve this.

**The learned model has not seen real data:** its mechanics were tested
with synthetic examples, improving accuracy from 50% to 100% in a
constructed case. Real-world performance remains unknown. Fixed rules
remain in use until enough ratings are available.

**Automatic retry has no demonstrated benefit yet:** it is implemented and
tested, but no case in the calibration set improves with retry. All failures
in that set are tracking problems, which retry correctly skips.

---

## Tested and rejected

Recorded so the same unsuccessful approaches are not repeated:

| Idea | Result |
|---|---|
| Divergence as a motion-center estimator (Funscript Flow) | 41 px instead of 110; correlation 0.829 versus 0.904/0.917. Makes the combination worse. |
| Symmetric projection weighting (Funscript Flow) | No measurable difference: 110.5 versus 110.1. |
| Integrated optical flow as a fusion input | Correlation 0.27–0.78; amplitude off by up to 2.4×. Integration accumulates estimation errors. |
| Camera correction using the median flow field | Corrects vectors rather than positions; ineffective for this backend. Replaced with feature-based position correction. |
| Fusion of CSRT and flow backend | Always falls **between** the sources, never beats the best: clean 1.000/0.926 → 0.985; occluded 0.293/0.747 → 0.620. Agreement between two sources is symmetric: it reveals disagreement, not which is right. `fusion.py` is tested but not connected. |
| Minimum cycle count as a periodicity criterion | Penalizes slow but valid motion: two clean cycles would appear unreliable. **Active time fraction** separates cases better: one excursion 0.27, two slow cycles 0.86, continuous motion 1.00. |
| Measuring rhythm globally across a video | Assumes a single continuous rhythm. Two real scripts for the same 77 s film scored 0.203 and 0.039 globally, classifying **both** as noisy, including one from an established tool. Windowed measurement (8 s, median): 0.396 and 0.324. |
| Rhythm measurement without detrending | Linear drift scored 0.289, above the 0.20 rejection threshold, and would pass. A valid sine wave with drift fell from 1.000 to 0.425. Detrending fixes both. |
| Absolute RANSAC match count as a quality metric | Random matches on noise backgrounds increased amplitude to 233 px. The **fraction** separates cases cleanly: 0.24 versus 0.84–0.90. |
| Finer training ramps | Initially rejected for the wrong reason; actual resolution remains unknown. See the raw-value test. |
| CUDA through pip-installed OpenCV | `opencv-python` and `opencv-contrib-python` are built **without CUDA**. Requires a custom build or OpenCL instead. |

---

## Pitfalls

These issues have already cost development time:

- **Noise-background test videos do not work for camera compensation.**
  `goodFeaturesToTrack` finds no stable features, turning the estimate into
  a random walk and making compensation appear broken. Use textured backgrounds.
- **Moving objects in test videos must use world coordinates.** Objects
  drawn at fixed screen positions do not follow camera pans, making correct
  compensation appear wrong.
- **Increment `TRACK_CACHE_VERSION`** when changing `track_roi` or camera
  compensation, otherwise cached results silently come from the old code.
- **Regenerate Wails bindings** with `wails build` after adding a Go method.
  Frontend mocks are generated from `App.js`, so missing bindings surface quickly.
- **Measure rhythm only on the dense signal.** Peak/valley reduction turns
  any signal into a zigzag that appears rhythmic.
- **Build with `-trimpath`.** Otherwise the build machine's path, including
  the username, is embedded in the executable.

---

## Collaboration

`CONTRIBUTING.md` defines the workflow, boundaries, and central rule:
**behavior changes need a regression test verified to fail without the
change**. Documentation-only changes require factual checks.
GitHub runs `tests.yml` on every push and pull request (Go with the race
detector, Python, and frontend tests), and `release.yml` on version tags.

Previously, tests ran **only during releases**. With multiple contributors,
that would reveal problems only during delivery, long after the context
of another contributor's code had been forgotten.

## Tests

After completing the setup in `WIEDERAUFNAHME.md`, run the full suite (Bash):

```bash
go vet ./... && go test ./...
(cd generator && for t in *_test.py; do python3 "$t" || exit 1; done)
for t in cmd/gui-wails/frontend/test/*_test.py; do python3 "$t" || exit 1; done
```

Frontend tests load the real JavaScript in headless Chromium and replace
only the Wails bindings. Mocks are generated automatically from `App.js`
(see `test/_harness.py`).

---

## Why `fusion.py` and `videox` are not connected

Neither currently meets the rule from section 16 of the target concept:
no module without a demonstrably improved processing path. They remain in
the tree with tests because their prerequisites are foreseeable — a third
measurement source and removal of the Python dependency, respectively —
not because they are currently needed.

## Project status and next steps

The prioritized operational task list lives exclusively in
[docs/NEXT.md](docs/NEXT.md). Hardware and quality limitations above remain
open until measurement reports resolve them.

The release workflow ran successfully for `v0.2.1` on September 14, 2026,
publishing GUI and CLI binaries for Windows/Linux plus `checksums.txt`.
Tf/Tj, the second GUI region, and `suction_position` are integrated into
`main` through PRs #2–#4. This establishes build and integration status,
not performance on real hardware.

## Environment setup

Cloning, dependencies, and build instructions are maintained in
[WIEDERAUFNAHME.md](WIEDERAUFNAHME.md). Go and Wails versions follow
`go.mod`; CI and release builds use that same source for tool versions.
