clip_ausschnitt_b76a.mp4 — post step-2 eval (21 Sep 2026)

Generate: profile=standard, CSRT, ROI auto (511,72,684,576 ≈43% frame),
--contact-vibration --contact-vibration-curve soft

Results:
- Tracker: CSRT (no MIL) — OpenCV 5.0.0 contrib path OK in this env
- Signal Quality Doctor: 1.00 passed, no warnings
- Actions: 134 over ~50s, pos 0–100 mean 33.4
- device_recipe: sync=independent, contact_vibration=true, curve=soft  ✓ step2
- Playback map: contact ON maxVib=1.0; contact OFF maxVib=0.205
  (deep slice ~16% of keyframes pos>=75 get full contact envelope)

Caveats:
- Auto ROI is huge — not a careful mark; Motion Fidelity vs FunGen unknown
  (no reference provided for this clip)
- vibActiveFrames high even without contact = stroke intensity mapping
- Hardware feel not tested

Clip parked at uploads/ + tmp/clips/; funscript in this dir (not committed).
