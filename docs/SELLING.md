# Selling SamNPlayer — what we still need

Living checklist for turning the product into something people can
**find, trust, buy, and run**. Chat with the owner may be German; this
doc stays English with the rest of the product surface.

Related: `website/` (landing page source), `docs/CLOSED_SOURCE.md`,
`docs/LICENSE_SYSTEM.md`, `docs/COMPETITIVE.md`, `docs/PRODUCTION_ROADMAP.md`
stream E / C3.

---

## Verdict (today)

The app is close enough to demo and invite. It is **not** ready for
hands-off retail yet: license enforcement is off, there is no paid
checkout, and the public face is still a GitHub README rather than a
live Pages site.

---

## Must-have before charging money

| # | Need | Why | Status |
|---|------|-----|--------|
| 1 | **Public landing** with brand, one CTA, portable download | Buyers do not read repos | Source in `website/` — publish to `SamNPlayer-site` + Pages |
| 2 | **Production license keys** | Replace DEV issuer; keep private key offline | Infra built; **DEV pubkey** still embedded |
| 3 | **Enforcement ON** | Trial / unlicensed limits must match the offer | `license.Enforcement = false` |
| 4 | **Price + payment** | Stripe / Lemon Squeezy / similar → webhook → `license-tool issue` | Not built |
| 5 | **Key delivery** | Email the `SNP1` file after payment | Manual only |
| 6 | **Public binary channel** | Releases on `SamNPlayer-site` (or paid CDN) with checksums | Documented; publish script still missing |
| 7 | **Contact that works** | License requests, refunds, device help | Site CTA → showcase GitHub issue; swap to real inbox when ready |
| 8 | **Legal basics** | Proprietary license text, ToS, privacy (local-only claim) | MIT on old public copies; new work needs counsel if closed |

---

## Trust blockers (sell harder once fixed)

| Item | Buyer question it answers |
|------|---------------------------|
| Golden-clip / FunGen numbers on the site (short, honest) | “Is generation actually good?” |
| Windows portable smoke on a clean PC (no system ffmpeg) | “Will it run for me?” |
| Real Neo 2 clip of Play + contact vibration | “Does the device feel right?” |
| macOS build (later) | “Do you support my machine?” |
| Same English copy on README, site, and in-app | “Is this a real product or a hobby dump?” |

Do **not** invent unmeasured quality claims. Prefer one screenshot + one
number over a feature wall (`docs/COMPETITIVE.md`).

---

## Offer to lock (product)

Already decided in `docs/LICENSE_SYSTEM.md`:

- **Retail:** 1 person, 1 year, full Generate + `.samn` play  
- **Trial:** 1 minute Generate; Funscript play without Neo-2 `.samn`  
- **Invite / internal:** no expiry for testers  

Still to decide with the owner (not inferred here):

- Exact **EUR/USD** price and renewal wording  
- Refund window  
- Whether Discord / email is the support channel  
- Whether Anna-class reviewers get invite keys only (no public download)

---

## Distribution shape

```text
  SamNPlayer (private source)
        │  tag release
        ▼
  build Win/Linux (+ portable ffmpeg)
        │  publish assets + checksums
        ▼
  SamNPlayer-site (public)  ←  GitHub Pages = website/
        │
        ├─ Download CTA
        └─ License CTA → payment → SNP1 key → Settings import
```

---

## Suggested order of work

1. Publish `website/` to Pages; keep license CTA on showcase issues (or real inbox)  
2. Owner sets retail price + support channel  
3. Production `genkey`, embed pubkey, flip enforcement  
4. Minimal checkout → issue key (even if semi-manual at first)  
5. Trust pack: portable Win test + one Neo 2 feel note + optional bench number  
6. Only then: ads, influencers, store listings  

Until step 3–4, treat every “sale” as an **invite + manual key**.

---

## Explicit non-goals (for the first paid wave)

- Phone app store listing  
- Cloud account / sync  
- Multi-seat team plans  
- Funscript marketplace  

Keep the pitch: **local, measured, Sam Neo 2 first-class, small native GUI**.
