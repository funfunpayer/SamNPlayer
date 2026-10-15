# Owner decisions for 2.0 Next

## Answered

| Date | Question | Answer |
|---|---|---|
| 12 Oct | Paid product? | **No.** One program, free, donations. Price, supporter threshold, pass scope, payment provider and the licence pass (P7) are dropped |
| 12 Oct | Name | "SamNPlayer 2.0 Next" = the next major version of this program (`docs/EDITIONS.md`) |
| 12 Oct | Cut AI / batch from a "Free" edition later? | Dropped — new features ship to everyone |
| 15 Oct | Where the plan lives | In this repo, `docs/next/` (moved from `SamNPlayer-Next-Level`) |

## Open

| # | Question | Context | Recommendation |
|---|---|---|---|
| D14 | **Suite files in public releases** — *before the next tag* | `release.yml` builds Suite and attaches `SamNPlayer-suite-*` to the release of this public repo; v0.7.0 would be the first release with them | Never publish Suite files; CI only compiles Suite ([N1](specs/N1_RELEASES.md)) |
| D12 | **`SamNPlayer-Next-Level` repo** | It holds a copy of this code (PR #2, unmerged) and ChatGPT's review (PR #3). `ROADMAP_2026-10.md` §2 still calls it the home of Next Level | Develop 2.0 Next here; close PR #2, keep or archive that repo |
| D13 | **Release source** | Installed apps check releases of `funfunpayer/SamNPlayer`; the download page is `SamNPlayer-site` | Keep releases here while the repo stays; revisit only if it moves |
| D6 | **Suite:** keep AI training / golden benchmark internal, or ship them? | Owner 11 Oct: training belongs to the internal system | Keep internal until the training UI is user-ready; then decide per feature |
| D10 | **DEV licence key in history** | `license/testdata/issuer.ed25519` is in the public history; without a licence system it protects nothing | No action; never reuse it if keys ever return |
