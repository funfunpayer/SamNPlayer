# Licence model

**Owner decision 10 Oct 2026:** SamNPlayer is **freeware** with an EULA and
financed by **donations**. It is **not open source**. The licence-key system
(Ed25519 `SNP1` keys, trial limits, feature gates, `cmd/license-tool`) was
**removed** — every build is fully unlocked.

| Topic | Now |
|---|---|
| Price | Free (private, non-commercial use) |
| Terms | `legal/EULA.md` — shown once per EULA version on first start (Accept / Decline and quit); Settings → Licence shows it again; shipped as `EULA.txt` |
| Third-party licences | `legal/THIRD_PARTY_NOTICES.md` — Settings → Third-party notices; shipped as `THIRD_PARTY_NOTICES.txt` (+ `FFMPEG_LICENSE.txt`) |
| Donations | `legal.PatreonURL` / `legal.PayPalURL` — Settings → *Support SamNPlayer* and the header ♥ Donate button (hidden until a URL is set) |
| Old keys | Ignored; nothing reads `license.key` any more |
| Source | Private repo; binaries on the public SamNPlayer-site releases (`docs/CLOSED_SOURCE.md`) |

Bump `legal.EULAVersion` whenever the EULA changes in a way users must accept
again. The EULA is a template written for this project, not legal advice —
have it checked by a lawyer before relying on it.

The pre-10-Oct key design is in git history (`license/`, last on main at
`533457e`).
