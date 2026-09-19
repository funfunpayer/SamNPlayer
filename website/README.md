# Public product site (source)

Static landing page for SamNPlayer. English product language
(`docs/LANGUAGE.md`). Visual tokens match the desktop GUI (dark ink,
gold + teal, Space Grotesk).

## Local preview

```bash
cd website
python3 -m http.server 8765
# open http://127.0.0.1:8765/
```

## Publish to the showcase repo

The public face is [`funfunpayer/SamNPlayer-site`](https://github.com/funfunpayer/SamNPlayer-site)
(no application source). After this tree is approved:

```bash
git clone https://github.com/funfunpayer/SamNPlayer-site.git
rsync -a --delete \
  --exclude README.md \
  website/ SamNPlayer-site/
# keep SamNPlayer-site SETUP.md / LICENSE; add index.html as GitHub Pages root
cd SamNPlayer-site
git add -A && git commit -m "site: product landing page" && git push
```

Then enable **GitHub Pages** on `SamNPlayer-site` (branch `main`, `/` root)
and set the repo homepage URL.

Media under `website/media/` is a mirror of `docs/media/` so images keep
working when the app repo is private (`docs/CLOSED_SOURCE.md`).

## Do not

- Point image `src` at `raw.githubusercontent.com/.../SamNPlayer/...` once
  the app repo is private.
- Turn the first viewport into a dashboard of stats or feature cards.
