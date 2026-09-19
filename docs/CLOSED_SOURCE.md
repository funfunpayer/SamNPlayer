# Closed source + public downloads

**Goal:** Anyone can **download releases** and see the product page.  
Nobody outside the team can **see or fork the source**.

“Anna” was a mix-up — this is for the **general public**.

## The right split

| What | Where | Visibility |
|------|--------|------------|
| **Source code**, CI, issues | `funfunpayer/SamNPlayer` | **Private** |
| **Landing page** (logo, screenshots, “why us”) | [`SamNPlayer-site`](https://github.com/funfunpayer/SamNPlayer-site) | **Public** |
| **GUI/CLI binaries + checksums** | **Releases on `SamNPlayer-site`** | **Public** |

### Important GitHub fact

If you only make `SamNPlayer` private and keep releases there, **anonymous
users cannot download** those assets. Public downloads must live on a
**public** repo (the showcase) or another public host.

```text
  [Private] SamNPlayer          build + tag vX.Y.Z
        │
        ▼
  CI builds Windows/Linux GUI+CLI
        │
        ▼
  [Public] SamNPlayer-site      gh release create vX.Y.Z  (same binaries)
        │
        ▼
  Anyone opens site → Downloads → gets ZIP/binaries, never the source
```

## Owner setup (one-time)

### 1. Make the app repo private

```bash
gh repo edit funfunpayer/SamNPlayer \
  --visibility private \
  --accept-visibility-change-consequences
```

GitHub → **Settings → Collaborators**: only the core team.

### 2. Keep the showcase public

https://github.com/funfunpayer/SamNPlayer-site  

Share **this** URL as the public face (not the private app repo).

### 3. Mirror screenshots into the showcase

After the app repo is private, raw image URLs from it break for visitors.
Copy media into the site repo (relative paths in its README):

```bash
git clone https://github.com/funfunpayer/SamNPlayer-site.git
mkdir -p SamNPlayer-site/docs/media
cp /path/to/SamNPlayer/docs/media/*.png SamNPlayer-site/docs/media/
cd SamNPlayer-site && git add docs/media && git commit -m "docs: mirror GUI media" && git push
```

### 4. Publish each version’s binaries on the **site** releases

After a private app tag builds (or after downloading from the private
release while you still have access):

```bash
# from a folder that contains the five release assets:
./scripts/publish-public-release.sh v0.5.8
```

Or manually:

```bash
gh release download v0.5.8 -R funfunpayer/SamNPlayer -D /tmp/snp-rel
cd /tmp/snp-rel
gh release create v0.5.8 -R funfunpayer/SamNPlayer-site \
  --title "v0.5.8" \
  --notes "SamNPlayer v0.5.8 binaries (closed source)." \
  SamNPlayer-gui-linux-amd64 \
  SamNPlayer-gui-windows-amd64.exe \
  SamNPlayer-cli-linux-amd64 \
  SamNPlayer-cli-windows-amd64.exe \
  checksums.txt
```

Repeat for every new version after the private release finishes.

## What does *not* work

| Idea | Why not |
|------|---------|
| Public app repo + “disable forking” | Clone / Download ZIP still gives **full source** |
| Private app repo + public expects downloads there | Release assets stay **login-gated** |
| MIT + “please don’t copy” | Already-public MIT copies remain usable by those who got them |

## License file

The repo root [`LICENSE`](../LICENSE) is a **proprietary** notice (not MIT):

- Source and private docs: no copy / redistribute without permission
- Official site binaries: personal non-commercial use allowed
- Past public MIT revisions are not undone (see the historical note in
  `LICENSE`); ask counsel if you need more

## Decision

**Source private · product page + release downloads public** via
`SamNPlayer-site`. Proprietary `LICENSE` on the app repo.
