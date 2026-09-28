# Virtual Person in SamNPlayer — für Endnutzer

**So einfach wie möglich:** Ordner ablegen → aktivieren. Kein Terminal nötig.

## In 3 Schritten

1. **Pack holen** — Virtual-Person-Ordner aus dem Animation-Studio-Release (enthält `samn-plugin.json`).
2. **Ablegen** — Ordner nach `%APPDATA%\SamNPlayer\plugins\` kopieren  
   *(oder in SamNPlayer: Einstellungen → Virtual Person → **Install pack…** / **Open Plugins folder**).*
3. **Aktivieren** — Einstellungen → Virtual Person → **Enable**.

Fertig. Create / Play bleiben unverändert.

## Wo ist der Plugins-Ordner?

| System | Pfad |
|--------|------|
| Windows | `%APPDATA%\SamNPlayer\plugins` |
| Linux | `~/.config/SamNPlayer/plugins` |
| macOS | `~/Library/Application Support/SamNPlayer/plugins` |

In der App: **Open Plugins folder** öffnet genau diesen Ort.

## Was muss im Pack sein?

```text
virtual_person/
  samn-plugin.json      ← Pflicht
  web/…                 ← Bühne (VRM-Viewer)
  assets/character-01/  ← Charakter / Props
```

Ohne gültige `samn-plugin.json` erscheint das Pack nicht in der Liste.

## Lizenz

Virtual Person ist im **normalen Jahres-Key (€40)** enthalten (Feature `virtual_person`) — kein Extra-Addon. Solange die Lizenz-Prüfung noch aus ist, kannst du Enable schon testen.

## Wenn etwas fehlt

- **Refresh** tippen, nachdem du den Ordner kopiert hast.
- Statuszeile zeigt `pack: none` → Ordner liegt falsch oder Manifest fehlt.
- `License required` → Key mit Feature `virtual_person` importieren (erst wenn Enforcement an ist).

Technische Vertrag / Handshake: [PLUGIN_SYSTEM.md](PLUGIN_SYSTEM.md).  
Pack bauen (Entwickler): Animation Studio `docs/samnplayer-plugin.md`.
