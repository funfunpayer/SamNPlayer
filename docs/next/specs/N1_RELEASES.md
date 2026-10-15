# N1 — Release safety and updater source

**Status:** spec, not built · before the next tag · 15 Oct 2026

## Today

| Piece | State |
|---|---|
| Editions | `edition` package, `-tags suite`; Suite-only bindings return `errSuiteOnly` (`requireSuite()` in `cmd/gui-wails`) |
| Updater | `update/update.go`: `RepoOwner/RepoName` = `funfunpayer/SamNPlayer`; asset kind `gui` / `suite-gui`, exact name match |
| Releases | `.github/workflows/release.yml`: on a `v*` tag builds Free **and Suite** for Linux + Windows and attaches **both** to the release |
| Public copy | `scripts/publish-public-release.sh` copies only Free files to `SamNPlayer-site` |
| Latest release | `v0.5.45` (2 Oct), no Suite files yet |

## Problem

This repo is public. The first release after 0.6.0 (v0.7.0 is next) would put
the internal Suite build on the public release page. Actions artifacts on a
public repo are downloadable by any logged-in GitHub user too, so "upload Suite
as an artifact" is not private either.

## Change

1. `release.yml`: keep the Suite build steps as a **compile check** (proves
   `-tags suite` still builds), but drop the Suite files from the release
   upload and from `checksums.txt`.
2. A guard step after packaging fails the job if any file to be published
   contains `suite` in its name.
3. The Owner builds Suite locally when needed (`docs/EDITIONS.md` commands).
4. Updater unchanged: Free keeps `gui`; a Suite build finds no `suite-gui`
   asset and reports "no update" (already the behaviour when the asset is
   missing — verify with a test).

## If releases ever move

If the Owner later moves releases (D13), installed apps still look at
`funfunpayer/SamNPlayer`. One bridge release here, whose binaries already carry
the new `RepoOwner/RepoName`, moves them over. Archive the old place only after
that.

## Tests

- Workflow lint (`actionlint`) on the changed `release.yml`.
- Guard step: a dry-run job (`workflow_dispatch`, no tag) lists the files it
  would publish and fails on any `suite` name.
- `update` test: Suite build with a release that has no `suite-gui` asset →
  no update offered, no error dialog.
