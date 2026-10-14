# SamNPlayer 0.7.0

This is the first release since v0.5.45. It also carries the work prepared for
0.5.46 and 0.6.0, which were never published. The full list is in
[`CHANGELOG.md`](../CHANGELOG.md).

## Release notes (for the GitHub release)

> **SamNPlayer 0.7.0: one player, five modes**
>
> **Five modes:** Player, Editor, Generator, Training and Scene Map, all in
> one window. The glowing Mode pill at the top switches between them; press
> M, then 1–5.
>
> **Open a video, get going.** Open… in the middle of the empty Player
> (or Ctrl+O, or drag and drop) takes a video or a script.
> - A video brings its script along when the script lies beside it with the
>   same name.
> - Without one, *Create script* takes the video to the Generator.
>
> **The Generator is an editor.** A new script stays right there under the
> video, ready to edit. Pick the device it is for. *Optimize for* prepares
> it for any device or for the Sam Neo 2.
>
> **Sam Neo 2 feel per scene.**
> - Suction follows the scene: *Oral*, *Intercourse*, *Breast* or *Hand*.
> - A mix slider moves the weight between vibration and suction, or
>   vibration can simply follow suction.
> - In Scene Map, every chapter gets a suggested feel. You confirm it; nothing
>   is applied by itself.
> - Leave all of it off and the classic feel is unchanged.
>
> **More devices.**
> - Vibrators can get their vibration from the video's sound.
> - TCode devices (OSR2 / SR6) connect over USB or Wi-Fi, with their extra
>   axes.
> - Strokers get a safety range, and they now ease in after a jump instead
>   of slamming.
> - Training and the Generator show only what your device can do.
>
> **Sturdier.** A Training pause is real now: the device rests at 0, and the
> clock and the timeline stop with it. A script with broken timestamps no
> longer runs the app out of memory. This release was checked against 758
> public community scripts.
>
> **Downloads:** `SamNPlayer-gui-…` (app), `SamNPlayer-portable-…` (app with
> ffmpeg, unzip and run), `SamNPlayer-cli-…` (command line). Checksums are in
> `checksums.txt`. The `SamNPlayer-suite-…` files are the internal edition.
> Do not link them publicly.

## After the tag: Owner checks on your own computer

Only the tag builds downloadable files. So push the tag first, then unzip
`SamNPlayer-portable-windows-amd64.zip` from the release and work through
the list one item at a time. Steps that need a real device are marked
**(device)**. If something is broken, take the release back: delete the tag
and the release, as described in the Owner guide. Claude fixes it, and you
tag again.

1. **Start:** the app opens without an update or error popup. The top bar
   shows **v0.7.0**.
2. **Open:**
   - Open… on a video that has a script beside it: the script plays with
     that video.
   - A video without a script: the question appears, and *Create script*
     opens the Generator with that video.
3. **Generator:**
   - Find the tip, then Create.
   - The new script opens under the video. Move a point, Undo, then save.
4. **Player:** play, pause, jump ±10 s and seek on the timeline. The curve
   and the video stay together.
5. **Scene Map:** a script with chapters shows its scenes. *Use* on one
   suggestion gives that card its feel badge.
6. **Training:** start, Pause (the clock stops and the device rests), Resume,
   End.
7. **(device) Sam Neo 2:** connect, play a script, Pause and Stop. Try a scene
   feel (Inspector → Devices) and feel the difference.
8. **(device) Stroker or TCode, if you have one:**
   - Seek far ahead while playing: the device eases there and does not slam.
   - Stop halts the device.

If something fails, write down what you did and send Claude the log
(Tools → Log → Open log folder).

## Order

1. Merge the release PR.
2. Follow [`docs/owner/owner-aufgaben.md`](owner/owner-aufgaben.md), part 2:
   push the tag `v0.7.0`, watch the Release run, and paste the notes above
   into the release.
3. Work through the checks above with the downloaded ZIP.
