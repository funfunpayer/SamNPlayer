# Emotion Script — User handbook

**Product:** SamNPlayer Emotion GUI · **Format:** `.samn` (Emotion Script) · Funscript = share/import only.  
**Audience:** Owner + everyday users. English product copy.  
**Last updated:** 26 Sep 2026 (after Look v3 lilac, GapHeal, AIWrite S0→S1 export).

Related engineering docs: `EVERYDAY_GENERATE.md`, `AI_SCRIPT_WRITER.md`, `CONTENT_SOURCES.md`, `AUDIO_WORKFLOW.md`.

---

## 1. What you get

| Piece | Meaning |
|-------|---------|
| **Create** | Video → tip track (CSRT) → Emotion Script (`.samn`) + companion `.funscript` |
| **Play** | Watch curve on video, Neo 2 feel, edit dots, markers |
| **Device** | Connect Sam Neo 2 / Handy / etc. |
| **Training** | Practice patterns (no video required) |
| **AI Train** | Label scenes / tip ROIs for *helpers* (not the stroke writer) |
| **Settings** | Models, Collect learning data, logs |

**Default stroke writer is classical CSRT.** AI never silently invents the curve.

---

## 2. Everyday Create (recommended path)

1. Open **Create** → choose a video.  
2. **Find tip area** (or paint the box). Optional: *Smarter tip find* if an AI ROI model is set in Settings.  
3. Leave **Contact vibration** on (default). Optionally mark nipples/mouth (gold / magenta) for feel later.  
4. Click **Create** / Generate.  
5. **Review & improve:** trim · **Fill gaps** · **Heal tracking gaps** · audio check.  
6. Open **Play** — soft curve + dots. Edit if needed. Connect device when ready.

Auto after Generate: fill gaps + heal known tracker-loss windows (linear bridge only).

---

## 3. FAQ

### Is everything in the GUI?
**Almost for everyday use.** Create, Play, Device, Training, Settings, Scene map (Advanced), learning export, classical AI-training export, Load project / Scale range, Optimize for Neo 2 — yes.

Still **CLI / advanced / not Everyday GUI:**
- Whole-frame **4-zone** stroke mode (kept for experiments; weaker on measured clips)
- Flow tracker as Generate default (CLI)
- Full AI **draft script** inference (S2+ — button present but disabled until a local model exists)
- Some OFS bookmark editors (API exists; prefer Play markers / chapters for now)

### Why is Create better than importing a random Funscript?
Create tracks the tip in *your* video. Import + **Optimize for Neo 2** polishes foreign scripts (fill gaps → contact → bake) but cannot invent missing tracking.

### What is Heal tracking gaps vs Fill gaps?
| Control | Does |
|---------|------|
| **Fill gaps** | Linear points across *long time holes* between actions |
| **Heal tracking gaps** | Uses `tracking_gaps` from Generate: strips junk *inside* loss windows, bridges **only those windows**, clears metadata so Contact vib is not muted forever |

Neither re-runs CSRT. Neither invents motion from audio (audio only spaces steps).

### Can AI write the Funscript?
**Path is open, opt-in, local only** (`docs/AI_SCRIPT_WRITER.md`):
- **S0** plan + stub — shipped  
- **S1** **Export classical run** (Advanced) — ships samples for offline training  
- **S2+** local model draft → Quality Doctor → Keep (not default)

Cloud “ChatGPT writes positions” is **out of product**.

### Smarter tip find / Suggest profile — do they change the curve?
No. They **propose** tip box or style. You Apply. Curve still comes from CSRT Create.

### Scene map?
Advanced → **Show scene map**: rhythm heatmap + marks (exclude/source/region). Explicit only — never auto before Create. Optional **Export for learning** needs Settings → Collect learning data.

### Contact vibration vs tip path?
Contact vib follows stroke depth by default. With **Record tip path** + contact marks, Play can also buzz when the tip grazes a mark (Feel Stage A).

### Mac / Linux / Windows?
Windows portable is the primary smoke target. See release notes for your build.

### Trial / license?
See Settings / license UI for the current seat rules (trial length may change by release).

---

## 4. Play — quick map

| Control | Use |
|---------|-----|
| Play / Stop | Device + curve (default). Video ▶ can also start device (toggle). |
| Edit curve | Drag dots; click empty = add; double-click = delete (≥2 remain). |
| Optimize for Neo 2 | Import path: fill/heal gaps → contact on → bake vibe/suction → `.samn` |
| Load / Save project | `.snp.json` — script, video, offset, seek, loop |
| Scale range ×0.8 | Soften marked range intensity |
| Heatmap / markers | Seek, loop, Extended-O, O-markers |
| Contact strength / curve | Live feel without rewriting file |

Keyboard: Space · ←/→ · ,/. · 1–9 · +/− offset · L loop · E Extended-O · O marker — see Play help chip.

---

## 5. Settings that matter

| Setting | Effect |
|---------|--------|
| AI ROI model path | Enables *Smarter tip find* |
| Collect learning data | Allows Scene map **Export for learning** |
| AI draft model path | Reserved for S2+ (draft stays disabled until Available) |

---

## 6. Files on disk

| File | Role |
|------|------|
| `*.samn` | Emotion Script (source of truth) |
| `*.funscript` | Community companion / export |
| `*.snp.json` | Playback project |
| `…/roi_training_dataset/scene_map_learning/` | Scene-map collect (opt-in) |
| `…/roi_training_dataset/ai_script_imitation/` | Classical runs for AI draft training (S1) |

---

## 7. Troubleshooting

| Symptom | Try |
|---------|-----|
| Flat / stuck curve | Re-find tip · Invert motion · Heal tracking gaps · Check tracking_gaps muted Contact |
| Feels inverted | Advanced → **Invert motion direction** |
| Camera pans drift | Camera motion compensation · Fix contact area (static) off |
| Long-clip drift | Advanced → **Rhythm-robust signal** (opt-in) |
| Audio “wrong tempo” | Warn only — fix ROI/axis; does not rewrite curve |
| AI draft greyed out | Expected until S2 model — use **Export classical run** for S1 |

---

## 8. What we optimized for (honest)

- **Reliable Everyday CSRT** over flashy AI curve writers  
- **Opt-in** AI helpers (ROI, profile suggest, scene map, future draft)  
- **GUI load rule:** shipped features must appear in Emotion UI  
- Look: dark ink + **lilac accent** + teal secondary (Owner preference over honey orange)

Not yet the “best possible” recognition endgame — that needs measured gates (Owner: speed-cap, ≥4–5 rhythm clips, G1.3, AI S2 metrics). This handbook matches **what ships today**.
