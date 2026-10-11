# Flow look consolidation — 11 October 2026

## Decision and ownership

Owner asked to evaluate Grok's additions and remove unnecessary work.
Claude's S3 structure at b1862f4059ae6b994f434605e2b65ef55f507800
(PR #482) is the integration target. Claude owns core and structure;
ChatGPT owns this isolated visual layer. No pushes to claude/*.

## Disposition of the earlier work

| Contribution | Disposition | Reason |
|---|---|---|
| Flow navy/cyan/orange palette and channel tokens | Reuse #480 | Already implemented in the base stylesheet |
| Grok-attributed #481 phase 10: focus, contrast, reduced motion, overlay surfaces | Consolidated into theme_flow.css | Useful visual improvements, scoped to actual components |
| #481 phase 11 overlay polish | Consolidated | Existing tokens; no duplicate stacking or visibility rules |
| #481 broad body, forms, buttons, tables, lists, scrollbar and palette overrides | Omitted | Competing global style layer; can undo component semantics and established theme |
| #481 empty-state display rules and old rail sizing | Omitted | Visibility bugs and incompatibility with the horizontal shell |
| #481 flow_workspace.js and its main.js/index.html integration | Not carried forward | Competing sidebar mechanism; retain Claude's existing controls |
| Grok S3 structural commits on Claude's branch | Not imported | Claude reports recovery/reverts and an ours merge in #482; branch history remains |
| Earlier PR #481 | Superseded by this PR | Preserve discussion and branch history; close the duplicate proposal |

Grok attribution is supported by the phase 10 PR comment; the GitHub
account is shared, so individual authorship of every other edit is not
asserted. This is a review of the actual code, not the phase summaries.

## Integration

Only new files are proposed; no existing production file is replaced.
Claude imports `src/theme_flow.css` once, after `style.css`, when integrating
S3. The new stylesheet changes surfaces, interaction colours, keyboard
focus and accessibility preferences. It adds no JS, device calls, axes,
sample media, background animations, positioning or canvas dimensions.
Until imported, it has no effect in the running app.

## Verification completed

- Parsed all 18 selector rules and 33 declarations, including media rules,
  with tinycss2; selectors compiled with cssselect2.
- Checked that the visual layer contains no display, position, width,
  height, min/max-height, grid-template-columns, order or z-index overrides.
- Calculated contrast on base surface #0a1b2b:
  muted #9db8ca = 8.42:1; text #e9f7ff = 15.95:1;
  accent #00d9f4 = 10.15:1. These are specific colour-pair checks,
  not whole-app accessibility certification.
- Browser/app/CI validation is still pending. The local Playwright browser
  download failed with invalid/truncated archives; no visual pass claimed.

## Acceptance after S3 integration

- At 1366×768 and 1920×1080, inspect Player empty and loaded states without
  a sample video. Check keyboard focus on Library, Inspector and bottom tabs.
- Switch Player → Generator → Training → Tools → Player: only the active
  main panel may be visible. Inspector and bottom-panel visibility must
  remain controlled by the original code.
- Check three channel labels/colours and live values; no invented controls.
- Check help, handbook, error toast and drop overlay opening/closing;
  focus and stacking remain correct.
- Enable reduced motion and higher contrast; no animated overlay entrances,
  and active tabs remain distinguishable by underline.
- Verify existing frontend tests, S3 player_layout_test and build.
- Scene images/markers remain Generator-first; Player off by default per
  the latest Owner decision in #482.

## Findings handed to Claude in #482

1. S3 `#tab-playback[data-pb-layout="v1"] { display:grid; }` lacks an
   active-tab guard. It can outrank the inactive .tab-panel display rule.
   Verify with the real stylesheet; guard the active grid rule or add an
   explicit inactive rule in Claude's owned stylesheet.
2. PLAY_UI_CONTRACT.md still contains pre-V1 feature statuses (Recent,
   speed, auto-sync) and lacks the S3 hook inventory described in the PR
   comment. Reconcile it with the newer V1_PLAN.md.
