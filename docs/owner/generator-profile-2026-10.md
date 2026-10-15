# Generator profile, 15 Oct 2026 (migration step M0)

**Question:** where does Everyday Generate (CSRT) spend its time, and what
does the move from Go to C++ and Python gain?

**Tool:** `scripts/profile_generate` (build tag `opencv`). It runs
`GenerateNativeCSRT` under a Go CPU profile. Each OpenCV call appears as its
own `_Cfunc_…` entry. `-synth WxH@fps:seconds` writes a synthetic test clip
first: a white box moving up and down at 1.5 Hz on a noisy test pattern, so
the true motion is known and no real footage is needed.

```text
go run -tags opencv ./scripts/profile_generate -synth 1920x1080@30:60
go run -tags opencv ./scripts/profile_generate -video clip.mp4 -roi x,y,w,h
go tool pprof -top -nodecount=30 cpu.out
```

## Result: synthetic 1080p, 30 fps, 60 s (1800 frames), 4 cores

Total time is **174 s, about 2.9× slower than real time**. The curve
follows the true motion with r = 0.94 (|r|; the sign is flipped because
image y points down).

| Part (main thread) | Time | Share | Language today |
|---|---|---|---|
| CSRT update (`Tracker_Update`) | 81 s | 46 % | C++ (OpenCV contrib) |
| Camera shake estimate (`EstimateCameraMotionYExcludes`: features + LK flow + RANSAC, every frame at full size) | 71 s | 41 % | C++ (OpenCV) |
| Scene-cut check (`SignatureDiff`, `HistCorrelation`) | 17 s | 10 % | C++ (OpenCV) |
| Frame read on the main thread | 4 s | 2 % | C++ (OpenCV / FFmpeg) |
| **Go code** (control flow, post-processing, writing) | **< 0.1 s** | **≈ 0 %** | Go |

Decoding uses another 53 s of CPU, in FFmpeg's own threads alongside the
main thread.

**Conclusion:**
- About 99 % of the time is already spent in C/C++ (OpenCV, FFmpeg).
- Replacing Go does **not** make Generate faster by itself.
- Speed has to come from changes to the C++ work: what runs, at which
  size, and on how many threads.

## Experiment: camera estimate on a half-size picture

The camera shake was estimated on a picture at most 960 px wide, and the
shift was scaled back.

| | Time | r against the true motion |
|---|---|---|
| Before | 174 s | 0.94 |
| Half-size camera estimate | 106 s (−39 %) | **0.71** |

**Rejected:** much faster, but the curve got worse, so it fails the quality
gate. It was not kept.

## Next candidates (C++, each behind the gate: equal r, then time)

1. **Camera estimate on a second thread.** It needs only the previous and
   the current frame, so it can run alongside CSRT. The result is the same
   by construction; expected up to about −40 % wall time on 4 cores.
2. **Scene-cut check on a small picture.** It is a histogram and a
   signature, so size barely matters (10 %).
3. **CSRT on a cropped region** around the box instead of the full frame;
   measure the quality first.

## Still needed

These numbers come from a synthetic clip. The real ranking may differ, for
example because of more camera motion or longer scenes. Needed: the same
run on the Owner's golden clips (`clip_ausschnitt`, `clip_voll`). They are
not in the cloud workspace and must stay out of the repo. The Owner can
run the command above on their PC, or upload the clips for one session
only.
