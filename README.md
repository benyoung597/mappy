# omarchy-mappy

Remap a QMK keyboard without opening a browser.

ZSA's Oryx is a good web editor, but changing one key means logging into a
website, dragging it around, and flashing through their GUI. The layout is
open source and the toolchain is local — none of that needs a browser.

mappy is a CLI that reads a keymap, edits it, compiles it, and flashes it,
plus (eventually) an Omarchy 4 shell overlay that draws the board and drives
the CLI.

Status: **spike.** The build and flash round trip is verified end to end on a
Moonlander Mark I rev A, and the read/write data model is settled. The CLI can
locate the layout inside a `keymap.c` and nothing more yet.

## Layout

```
cli/     the tool — keymap data model, compile, flash.  See cli/README.md
plugin/  Omarchy 4 shell overlay, kinds: ["overlay"].   Not started
```

## Getting started

There is nothing to install yet. [`cli/README.md`](cli/README.md) documents the
whole build and flash pipeline as verified by hand — the toolchain, ZSA's fork,
the patch Arch needs, board revision detection, and where a layout actually
lives. Following it gets you building firmware from source without mappy at
all, which is the point: the tool automates a process that works on its own
first.

## Design

Two constraints drive most of it.

**Keyboards beyond the Moonlander are close to free.** `qmk info` returns real
physical key coordinates for every board in QMK, so a renderer that reads
that data is generic by default rather than by effort.

**The layout must be edited in place, not regenerated.** A `keymap.c` holds far
more than the key grid — custom keycodes, RGB code, tap-hold tuning — and QMK's
own `json2c` discards all of it. mappy replaces the byte range holding the
`keymaps[]` array and leaves every other byte untouched.

## Planned features

Nothing here is designed or committed to — a parking lot so ideas do not get
lost.

- **Typing heatmap**, like Keymapp's. Enable, disable, reset, and show
  per-key press counts over the board diagram.

  Worth recording now: the layout already builds with `ORYX_ENABLE = yes` and
  the `zsa/oryx` module, which is exactly what lets Keymapp see live key
  events over HID. That stream reports **matrix position and active layer**.
  Reading keys from evdev or the compositor instead would be easier but only
  yields the resulting keycode — no layer, no matrix position — which is not
  enough to colour a key on a diagram when the same keycode appears on three
  layers. If the heatmap gets built, the HID route is the one that works, and
  the firmware side of it is already enabled.
