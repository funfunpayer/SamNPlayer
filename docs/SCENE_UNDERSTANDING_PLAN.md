# Scene understanding — what is what, what moves against what (plan)

Status: **stage 2 tools done, measured; stage 2b (VLM clip mode) built,
waiting for an Owner PC run** (Claude, 28 Sep 2026; results in §3
"Stage 2 — results"). The hybrid gap-filler and the stable runtime (§4) are
next. The Owner approved:
*"Yes, definitely — plan it, put it on the board with Cursor, start Stufe 2.
Video recognition would be good too. The stable system without big AI comes
next."*

Builds on:
- `docs/VLM_TEACHER_PLAN.md` (teachers, contact points, own RF-DETR model)
- `docs/SCENE_MAP_PLAN.md` (marks, learning export)
- `docs/TFTJ_PROFILE_DIRECTION.md` (profiles, primary stroke, contact partner)

---

## 1. Why contact points are not enough

A contact point says only **where** the stroke is. It fixed the
multi-person clip (r 0.304 → 0.406), but the engine still does not know:
- **what** each region is: body parts, and which person they belong to;
- **what moves against what**, the moving part and its partner — for
  example mouth ↔ penis, hand ↔ penis, breasts ↔ penis, vagina ↔ penis;
- **what kind of scene** it is, and **how** it moves: axis, depth range,
  tempo, pauses.

That is the information that decides **what gets tracked**:
- the primary stroke target,
- the contact partner for contact vibration,
- what to ignore,
- which profile to use.

Today a person sets all of this by hand.

## 2. Principle — hybrid, the Owner's rules kept

- **The AI proposes, the classical engine measures, the user applies.**
  The rules in `TFTJ_PROFILE_DIRECTION.md` stay:
  - proposals for the primary stroke and the contact partner;
  - silently committing ROI2 remains forbidden;
  - the AI never invents the 0–100 curve.
- Proposals use the types that already exist: `ROICandidate` with `Class`
  for "pick primary" (TFTJ 4b), and `RegionClass` / `RegionClass2` on
  ROI / ROI2.
- The profile proposal is the "auto profile from the scene pattern" the
  Owner asked for.

## 3. Stages

| Stage | Teachers deliver | The system does with it | Status |
|---|---|---|---|
| **1** | contact point | the rhythm grid searches there | done (#312) |
| **2** | body-part boxes plus **roles**: moving part, partner, ignore | proposals for the primary stroke ROI, the contact partner ROI2 and excludes | **tools done** (opt-in; GUI = Cursor) |
| **3** | scene type plus axis, depth and tempo per shot | profile proposal per shot; depth range for the mapper | next |
| **4** | the course over time | draft script from our own model (hybrid: engine curve plus model structure), through QD and Keep | later |

### Stage 2 — mechanism

1. **Parts (what is what)**, per keyframe, from any teacher:
   - NudeNet: all classes, mapped to the canonical ids in
     `bodyparts.py` / `BODY_REGIONS.md`.
   - A VLM probe: boxes by label.
   - Our RF-DETR, which is multi-class; the dataset builder already keeps
     every class.
2. **Motion (what moves)** comes from the engine itself, not from a big
   model: the rhythm-grid **scan** (`ScanSceneMap`, a per-window
   cell-score heatmap at the stroke tempo).
   - New CLI `scan-scene-map VIDEO --windows N` writes it as JSON.
3. **Roles by rule** (`scene_roles.py`):
   - Per window, a part's motion is the mean of the strongest third of
     the cells under its box (a plain mean dilutes large boxes such as
     breasts).
   - Pairs: `penis`/`glans` against `mouth`, `hand_*`, `breasts`,
     `vagina`.
   - Within the most-moving pair, the part that moves more is the
     **primary stroke target** (tracked) and the other is the **partner**
     (contact vibration target).
   - If the penis is hidden (in the mouth or between breasts), the partner
     part alone plus its motion gives the scene type.
   - Other people's parts and thighs become **exclude** proposals.
4. **Scene type from the pair:**

   | Pair | Scene type |
   |---|---|
   | mouth + penis | blowjob |
   | hand + penis | handjob |
   | breasts + penis | titjob |
   | vagina + penis | penetration |
   | hidden penis | from the partner part and its motion |

5. **Output: `<clip>.scene.json`.** Per window: parts, roles, scene type
   and confidence. Plus `proposals`: `ROICandidate`-shaped primary and
   partner boxes for the first shot and per shot, and a profile
   suggestion.
6. **Into the app:**
   - Go `LoadSceneProposals(path)` returns the proposals (primary and
     partner as `ROICandidate` with `Class`, scene type, confidence,
     window); `.At(ms)` picks the one for a video time.
   - The CLI `generate --scene-proposals FILE [--scene-at-ms MS]` uses the
     primary as ROI (and its class as region class when canonical) only
     when `--roi` is not given. It is explicit opt-in and logged. The
     partner is only logged — it is never applied as ROI2 (the CLI has no
     ROI2, and the rule stays until the Owner changes it).
   - The GUI shows them in the existing pick-primary flow and the user
     applies. That GUI part is Cursor's.

### Stage 2 — results (28 Sep, NudeNet as the only teacher)

Pipeline: `samnplayer scan-scene-map clip.mp4 --windows N --out scan.json`
→ `python3 generator/scene_roles.py --video clip.mp4 --scan scan.json
--nudenet --out clip.scene.json --contact-out clip.contact.json`.

| Clip | Anchor | r (FunGen ref) |
|---|---|---|
| multi-person 642 s | none (today) | 0.304 |
| multi-person | NudeNet contact points (stage 1) | 0.401 |
| multi-person | hand-labelled contact points | 0.425 |
| multi-person | **stage-2 moving part, confidence ≥ 0.5** | **0.440** |
| multi-person | stage-2 moving part, every typed window | 0.449 (but voll drops 0.752 → 0.702) |
| multi-person | stage-2 pairs only | 0.341 |
| clip_voll / clip_aus | stage-2, confidence ≥ 0.5 | unchanged (0.456/0.752, 0.482/0.888) |

- The moving part beats the teachers' contact point: it tracks **what
  moves**, not just what is visible.
- Scene type is right in 15 of 21 typed windows (21 of 27 windows typed).
  The typical miss is titjob read as blowjob when the face bobs with the
  stroke. Stage 3 (axis, tempo) and the VLM clip mode should separate
  these.
- The confidence floor 0.5 is what keeps the goldens unchanged: low-
  confidence windows would move a box that is already right.

## 3b. Modes — how much the AI does (Owner, 28 Sep)

The Owner wants several modes. All three use the same parts:

| Mode | What the AI does | What the engine does | Status |
|---|---|---|---|
| **Classic** | proposes ROI, partner, profile; the user applies | tracks and writes the curve | stage 2 (now) |
| **Hybrid gap-filler** | steps in only where the engine is weak: tracking lost, low confidence, QD flag, or nothing found; there it moves the search (contact points / moving part) or fills a draft segment | everything else | next after 2b: gap detection from QD + confidence, fill with stage-2 anchors first, stage-4 draft later |
| **AI script** | writes a draft script (structure: strokes, pauses, depth) from our own model | supplies the precise timing; QD checks; the user keeps or rejects per segment (Keep) | stage 4 |

**ROI2 — why it is never set automatically.** The locked rule in
`TFTJ_PROFILE_DIRECTION.md` is "no silent ROI2": a wrong contact partner
makes the contact vibration wrong, and the user may not notice. The
proposals already contain the partner. **Open Owner decision:** turn the
rule into a setting "Apply AI setup automatically" (default off, every
applied value logged and shown). Until the Owner decides, the partner stays
a proposal.

### Video, not just images

- **The engine is already video.** Optical flow and the rhythm grid see
  motion over time. Stage 2 uses exactly that for "what moves", so no big
  model is needed for it.
- **VLM clip mode (stage 2b, built):** `vlm_probe.py --clip N
  --clip-span-s S` sends N frames spanning S seconds (ending at the
  keyframe, oldest first) as one multi-image request. Qwen2.5-VL and
  Qwen3-VL are trained on video input.
  - It asks for the scene type, the moving part, its partner, the motion
    axis and the number of people, plus boxes for the last frame.
  - `scene_roles.py --vlm clip.vlm.json --truth
    generator/testdata/vlm_labels/multi_person_642s.scene_types.json`
    records the model's reading per window (`vlm`), counts agreement with
    the rules (`summary.vlm`) and scores both against Claude's hand
    labels (`summary.truth`; the rules alone score 15/21).
  - It does **not** change roles yet. Once a run on the Owner PC shows
    the VLM beats the rules on scene type, it becomes the tie-breaker
    (for example titjob vs blowjob when the face bobs).
  - Owner PC run: `python3 vlm_probe.py --video db.mp4 --model
    qwen2.5vl:7b --clip 6 --every-s 20`, then send `db.vlm.json` back.
- **Real action-recognition models** (a VideoMAE-style video classifier)
  come later. They need our own labelled shots first, which stages 2–3
  collect (confirmed scene types per shot).

## 4. The stable system without big AI (next, as the Owner asked)

At run time, only two small, fast parts that run everywhere:
1. **Our own RF-DETR multi-class detector** (ONNX via onnxruntime on CPU,
   CUDA or DirectML): what is what, per frame.
2. **The classical engine** (CSRT + rhythm grid + two-point): what moves,
   and how.

Stage 2's rules join the two: roles, scene type and proposals. Nothing at
run time needs Qwen, Colibri or NudeNet. Those remain optional
**teachers**; they produce training labels for (1) through the loop in
`docs/VLM_MODELS.md`.

Order of work:
1. Stage 2 with any teacher. NudeNet first, because it runs here and is
   measurable now.
2. Train (1) with the classes `penis`, `glans`, `mouth`, `hand`,
   `breasts`, `vagina`, `face`, `thigh`, `contact`.
3. Switch the part source to (1).
4. Measure on ≥ 4–5 clips.
5. Only then talk about defaults.

## 5. Measurement (every stage)

- **Roles:** on the labelled clips, is the proposed primary box the
  labelled contact region? This is the contact hit from `vlm_score.py`,
  per role.
- **Scene type:** accuracy against a hand label per shot. Claude labels
  the three clips we have.
- **End to end:**
  - Generate with the proposals applied, compared with today's default
    ROI.
  - On the multi-person clip, today's centre ROI sits on the heads, so
    the gain should show there.
  - The goldens must not regress.

## 6. Ownership

| Who | Does |
|---|---|
| Claude | `scan-scene-map` CLI, `scene_roles.py`, `LoadSceneProposals`, CLI opt-in, VLM clip mode, all measurements, stage 3 rules |
| Cursor | GUI: show the proposals in pick-primary / ROI2 (user applies), profile suggestion chip, candidate accept/reject (from V3), Settings AI setup buttons |
| Owner | GPU runs (VLM clip mode, RF-DETR training), 4–5 clips, confirming candidates |
