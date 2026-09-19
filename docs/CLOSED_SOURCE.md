# Closed source + public showcase

Anna (and anyone outside the core team) should **see the product**, not
**fork the source**.

## Setup (owner does this once in GitHub)

### 1. Public showcase (already created)

- Repo: https://github.com/funfunpayer/SamNPlayer-site  
- Contents: product landing (`website/` in this repo → sync) + branding (no application code)  
- Share **this** link (or GitHub Pages once enabled) with Anna / reviewers  
- Selling checklist: [`docs/SELLING.md`](SELLING.md)

### 2. Make the application repo private

In a terminal logged in as the repo owner (`funfunpayer`):

```bash
gh repo edit funfunpayer/SamNPlayer \
  --visibility private \
  --accept-visibility-change-consequences
```

Or: GitHub → **SamNPlayer** → **Settings** → **Danger Zone** →
**Change visibility** → **Private**.

Then: **Settings → Collaborators** → add only the core team.

### 3. After going private — mirror screenshots into the showcase

Raw image URLs from `SamNPlayer` stop working for the public once the app
repo is private. Copy `docs/media/*.png` into `SamNPlayer-site/docs/media/`
and point the showcase README at relative paths again.

```bash
git clone https://github.com/funfunpayer/SamNPlayer-site.git
mkdir -p SamNPlayer-site/docs/media
cp docs/media/*.png SamNPlayer-site/docs/media/
# edit README image srcs back to docs/media/...
cd SamNPlayer-site && git add -A && git commit -m "docs: mirror GUI media" && git push
```

### 4. Binaries for trusted people

Keep releases on the **private** app repo (or send ZIP builds privately).
Do **not** expect anonymous downloaders on the showcase page.

## What does *not* work

| Idea | Why not |
|------|---------|
| Keep repo public, only “disable forking” | Anyone can still **clone / Download ZIP** the source |
| MIT “but please don’t copy” | Already published MIT copies stay legal for people who got them |
| Public source + private “secret” folder | Git history still exposes it |

## Legal note

Code that was already public under **MIT** may have been cloned by others.
Making the repo private stops **new** public access; it does not erase past
copies. For a clean closed-source future: private repo + proprietary license
on new work; ask counsel if you need stronger protection.

## Decision (ROADMAP)

This replaces the open “open source or not?” question with:
**source private for the team; showcase public for visibility.**
