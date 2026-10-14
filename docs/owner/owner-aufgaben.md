# Was nur du machen kannst (Owner-Aufgaben)

Claude arbeitet in einer Cloud-Umgebung, die nur auf den eigenen
Arbeits-Branch schreiben darf. Ein paar Dinge gehen deshalb nur von deinem
Rechner oder über die GitHub-Webseite. Hier steht jeweils Schritt für
Schritt, wie.

Stand: 14. Oktober 2026.

---

## 1. Alte Branches löschen

Die Datei [`old-branches.txt`](old-branches.txt) listet **92 alte Branches**.
`main` und Claudes aktueller Arbeits-Branch stehen nicht drin.

**Warum das gefahrlos ist:**

- 80 Branches haben eine gemergte oder geschlossene PR. Ihre Commits bleiben
  über die PR auf GitHub erreichbar.
- 2 Branches stammen aus #489 und #521, die heute geschlossen bzw. gemergt
  wurden.
- 10 Branches haben keine PR. Claude hat sie geprüft:
  - Ihr Inhalt ist schon auf `main`: `claude/d3-device-ui`,
    `claude/import-theme-flow-20261011`, `feat/timeline-ruler-markers`,
    `cursor/ofs-inverted-ui`, `cursor/pr456-frontend-ui`,
    `cursor/ai-train-tags-track-80c5`.
  - Sie würden über 3000 Zeilen löschen (kaputte Platzhalter):
    `chatgpt/play-layout-flow-20261011`, `grok/play-layout-v2-flow-20261011`.
  - Sie sind veraltet: `cursor/cleanup-land-627e` (Stand v0.5.40, ohne
    gemeinsamen Verlauf mit `main`) und `grok/s5-transport-look-polish`
    (48 Zeilen CSS, vom neuen Look ersetzt).

### Variante A: im Terminal (schnell)

Im Repo-Ordner auf deinem Rechner:

**Windows (PowerShell):**

```powershell
git pull
Get-Content docs/owner/old-branches.txt | ForEach-Object { git push origin --delete $_ }
```

**Linux / macOS / Git Bash:**

```bash
git pull
xargs -n 20 git push origin --delete < docs/owner/old-branches.txt
```

Wenn ein Branch schon weg ist, meldet git „remote ref does not exist“. Das
kannst du ignorieren.

### Variante B: auf github.com

Repository → **Code** → **Branches** → beim jeweiligen Branch auf das
Mülleimer-Symbol klicken.

### Damit es nicht wieder voll wird

Repository → **Settings** → **General** → bei *Pull Requests* den Haken
**„Automatically delete head branches“** setzen. Dann löscht GitHub den
Branch einer PR, sobald sie gemergt ist.

---

## 2. Release veröffentlichen (Tag pushen)

Claude bereitet den Release vor: Versionsnummer, CHANGELOG und
Release-Notes in einer eigenen Release-PR. Den Tag kann nur du pushen. Der
Tag startet den Build (`.github/workflows/release.yml`): Windows und Linux,
Free- und Suite-Edition, mit ffmpeg.

1. Die Release-PR ist gemergt und die CI auf `main` ist grün.
2. Auf deinem Rechner:

   ```bash
   git checkout main
   git pull
   git tag v0.7.0
   git push origin v0.7.0
   ```

   Die Versionsnummer muss genau zu `VERSION` im Repo passen. Der Workflow
   prüft das als Erstes und bricht sonst ab.
3. Unter **Actions → Release** den Lauf beobachten (etwa 10 bis 20 Minuten).
4. Unter **Releases** sind dann die ZIP-Dateien zu finden. Die Release-Notes
   aus [`docs/RELEASE_0_7_0.md`](../RELEASE_0_7_0.md) (der eingerückte Block
   oben) einfügen, falls sie noch fehlen. Die `SamNPlayer-suite-…`-Dateien
   sind die interne Edition: nicht öffentlich verlinken.
5. **Durchklicken (etwa 15 Minuten):** die Portable-ZIP auf deinem
   Windows-Rechner entpacken, starten und die Checkliste in
   [`docs/RELEASE_0_7_0.md`](../RELEASE_0_7_0.md) („After the tag“) abarbeiten:
   Start, Video öffnen, Generator, Player, Scene Map und Training. Mit Gerät
   zusätzlich Sam Neo 2 und Stroker/TCode (Hardware-Test unten). Wenn etwas
   hakt, schick Claude das Log; der Tag lässt sich zurücknehmen (siehe unten).

**Falls etwas schiefgeht:** Den Tag löschen (`git push origin :v0.7.0` und
`git tag -d v0.7.0`), Claude den Fehler aus dem Actions-Log geben, den Fix
mergen und den Tag neu setzen.

---

## 3. Hardware-Test (nur mit echtem Gerät)

Claude kann nur gegen Test-Stubs und das Mock-Gerät prüfen. Mit echter
Hardware bitte einmal durchgehen:

**Sam Neo 2**

- [ ] Verbinden (oben rechts **Connect**) → grüner Punkt, Name „Sam Neo 2“.
- [ ] Player: Test-Clip mit Script abspielen → Gerät folgt, **Stop** stoppt sofort.
- [ ] Editor: **Optimize for → Sam Neo 2** → Vibration und Saugen spürbar.
- [ ] Training: beide Ringe sichtbar.
  - [ ] **Start**, dann **Pause** → beide Kanäle sofort aus.
  - [ ] **Fortsetzen** → es geht weiter.
  - [ ] **Interrupt** → der Zyklus endet.
  - [ ] **End** → alles aus.

**Stroker (The Handy, OSR2 / SR6), falls vorhanden**

- [ ] Verbinden → Player folgt dem Hub, Bereich Bottom / Top wirkt.
- [ ] Training: Hinweis „has no vibration or suction motor“, Start gesperrt.

**Allgemein**

- [ ] Modus-Pille oben: alle fünf Modi öffnen. Taste **M** öffnet das Deck.
- [ ] Generator: einen kurzen eigenen Clip erstellen, mit **Optimize for**
      einmal *Off* und einmal *Sam Neo 2*.

Was nicht passt, als Issue anlegen oder Claude direkt schreiben, möglichst
mit Screenshot und dem Log (Tools → Log).

---

## 4. Was Claude macht (zur Einordnung)

- Code, Tests, Screenshots, Doku, PRs und Merges nach grüner CI.
- Issues pflegen und schließen.
- Release vorbereiten: Version, CHANGELOG, Notes.
- **Nicht möglich:** fremde Branches löschen, Tags pushen, echte Hardware
  bedienen, Repository-Einstellungen ändern.
