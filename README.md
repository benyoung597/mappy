# Mappy

[![status](https://img.shields.io/badge/status-working%2C%20one%20keyboard-brightgreen?style=flat-square)](#)
[![go](https://img.shields.io/badge/go-1.27-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![platform](https://img.shields.io/badge/platform-linux-333333?style=flat-square&logo=linux&logoColor=white)](#)
[![qmk](https://img.shields.io/badge/qmk-zsa%20fork-ff6600?style=flat-square)](https://github.com/zsa/qmk_firmware)
[![tested on](https://img.shields.io/badge/tested%20on-moonlander%20mk%20I%20rev%20A-6b46c1?style=flat-square)](#)
[![license](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)

Remap a QMK keyboard without opening a browser.

ZSA's Oryx is a good web editor, but changing one key means logging into a
website, dragging it around, and flashing through their GUI. The layout is
open source and the toolchain is local — none of that needs a browser.

Mappy is a CLI that reads a keymap, edits it, compiles it, and flashes it,
plus (eventually) an Omarchy 4 shell overlay that draws the board and drives
the CLI.

Status: **working, for one keyboard.** The full loop — read a keymap, change a
key, build the firmware, write it to the board — runs from the terminal and is
verified end to end on a Moonlander Mark I rev A. Only single key edits exist;
there is no shell overlay yet.

## Layout

```
cli/     the tool — keymap data model, compile, flash.  See cli/README.md
plugin/  Omarchy 4 shell overlay, kinds: ["overlay"].   Not started
```

## Usage

```sh
make build   # builds ./mappy in the repo root
K=~/.config/qmk-userspace/keyboards/zsa/moonlander/keymaps/vimster/keymap.c

./mappy get -file $K 0 40          # DUAL_FUNC_0
./mappy set -file $K 0 40 KC_F5    # 0 40: DUAL_FUNC_0 -> KC_F5
./mappy compile -file $K
./mappy flash -file $K
```

Keys are addressed by index, deliberately: it keeps the CLI an honest machine
interface and pushes ergonomics into the UI where they belong. `mappy get`
with no index prints every layer so you can find the one you want, and
`-json` pipes it to `jq`.

`-keyboard` is detected from USB, and `-keymap` comes from the path, so
neither is usually worth typing. `mappy help` lists the rest.

## Getting started

[`cli/README.md`](cli/README.md) documents the whole build and flash pipeline
as verified by hand — the toolchain, ZSA's fork, the patch Arch needs, board
revision detection, and where a layout actually lives. Following it gets you
building firmware from source without mappy at all, which is the point: the
tool automates a process that works on its own first.

```sh
make test              # no toolchain needed
make integration-test  # + qmk c2json
make compile-test      # + a real firmware build
make flash-test        # + an attached keyboard, writes nothing
```

## Design

Four decisions drive most of it.

**Keyboards beyond the Moonlander are close to free.** `qmk info` returns real
physical key coordinates for every board in QMK, so a renderer that reads
that data is generic by default rather than by effort.

**The layout must be edited in place, not regenerated.** A `keymap.c` holds far
more than the key grid — custom keycodes, RGB code, tap-hold tuning — and QMK's
own `json2c` discards all of it. mappy edits bytes: `set` replaces the range
holding one keycode, so Oryx's column padding survives and changing a key is a
one line diff. Rewriting the whole array is a 234 line one.

**Nothing is written until qmk agrees it parses.** An edit goes to a temp file
beside the target, is read back through `c2json`, and is checked to hold the
new keycode and nothing else before it is renamed into place. A `keymap.c`
mappy cannot produce cleanly is a `keymap.c` you never see.

**The board is asked what it is.** `reva` and `revb` are separate targets with
different bootloaders, and cross flashing writes settings to storage the
hardware does not have. mappy reads the USB product id from `/sys` and refuses
to flash a target that is not the one attached.

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

## Contributing

Conventional commits, work from `development`, and releases come out of
release-please. [`CONTRIBUTING.md`](CONTRIBUTING.md) has the details.

## License

[MIT](LICENSE).
