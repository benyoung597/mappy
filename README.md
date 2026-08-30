# omarchy-mappy

Edit and flash a QMK keyboard layout without a browser. A CLI that owns the
JSON → compile → flash pipeline, and (later) an Omarchy 4 shell overlay that
drives it.

Status: **spike**. The build and flash round trip is verified end to end on a
Moonlander Mark I rev A, and the read/write data model is settled. No tool has
been written yet.

## Verified build process

Everything below was run on Omarchy 4.0.1 (Arch) against a Moonlander Mark I,
Oryx firmware revision v25.0.

### 1. Toolchain

```sh
sudo pacman -S --needed qmk arm-none-eabi-gcc arm-none-eabi-newlib \
                        arm-none-eabi-binutils dfu-util
```

`qmk doctor` reports a failure compiling with `avr-gcc`. Ignore it — that is
the 8-bit AVR toolchain, and the Moonlander is ARM (STM32F303). Only the
`arm-none-eabi-*` and `dfu-util` checks matter.

### 2. udev rule for flashing without root

```sh
sudo tee /etc/udev/rules.d/50-zsa.rules <<'RULE'
# STM32 DFU bootloader — lets the logged-in user flash without root
SUBSYSTEM=="usb", ATTR{idVendor}=="0483", ATTR{idProduct}=="df11", TAG+="uaccess"
RULE
sudo udevadm control --reload-rules && sudo udevadm trigger
```

`TAG+="uaccess"` grants access to whoever is logged in at the seat. No
`plugdev` group, and no need for ZSA's `wally-cli` from the AUR.

### 3. ZSA's QMK fork

An Oryx export will **not** build against mainline QMK. It needs the
`zsa/oryx` and `zsa/defaults` modules and `ORYX_ENABLE`, which only exist in
ZSA's fork.

The branch matches the firmware revision badge shown in Oryx (top right of the
layout page) — `firmware25` for "firmware v25.0". Branches `firmware17`
through `firmware25` exist.

```sh
git clone --branch firmware25 --recurse-submodules \
  https://github.com/zsa/qmk_firmware ~/qmk_zsa
```

### 4. Patch for newer GCC

ZSA's build servers use GCC 12 (visible in the Oryx `build.log`); Arch ships
16.2.0. The exact version where this starts failing is not known — only 16.2.0
was tested. `keyboards/zsa/moonlander/moonlander.c`
fails to assemble:

```
Error: Thumb encoding does not support an immediate here -- `msr msp,[r3]'
```

The inline asm constrains the operand with `"g"` (any general operand,
including memory), but `msr` only accepts a register. GCC 12 happened to pick a
register; GCC 16 picks memory. The constraint was always wrong.

```sh
cd ~/qmk_zsa
sed -i '495s/"g"/"r"/' keyboards/zsa/moonlander/moonlander.c
```

Line 495 on branch `firmware25`:

```c
__asm__ volatile("msr msp, %0" ::"r"(*(volatile uint32_t *)APP_ADDRESS));
```

This hits every Arch user building ZSA firmware, so the tool must carry it as
a patch — it lives in a tree we do not own and vanishes on re-clone.
Not yet reported upstream.

### 5. Keep the layout in an external userspace

The layout lives outside the firmware tree. QMK's external userspace exists for
exactly this, and it means `~/qmk_zsa` stays a disposable build dependency you
can delete and re-clone without losing anything.

A userspace is any directory with a `qmk.json` at its root:

```
<userspace>/
  qmk.json
  keyboards/zsa/moonlander/keymaps/vimster/{keymap.c,config.h,rules.mk,keymap.json}
```

```json
{
    "userspace_version": "1.1",
    "build_targets": [
        ["zsa/moonlander/reva", "vimster"]
    ]
}
```

`userspace_version` must be the string `"1.1"`; `build_targets` entries are
`[keyboard, keymap]` tuples. Schema:
`<qmk-tree>/data/schemas/user_repo_v1_1.jsonschema`.

QMK does not care where the directory is, so it can be a stow package in an
existing dotfiles repo rather than a repo of its own. This setup uses:

```
~/.dotfiles/qmk/dot-config/qmk-userspace/  ->  ~/.config/qmk-userspace/
```

**Do not stow `~/.config/qmk/`.** That is QMK's own config directory, and
`qmk.ini` stores `user.overlay_dir` and `user.qmk_home` as absolute paths —
machine-local by nature, and wrong on any other machine. The userspace is a
sibling of it, not a child.

Point QMK at both the userspace **and** the ZSA fork:

```sh
qmk config user.overlay_dir="$HOME/.config/qmk-userspace"
qmk config user.qmk_home="$HOME/qmk_zsa"
qmk userspace-doctor    # confirm both paths
qmk userspace-compile
```

Both settings are required. `qmk compile` picks up the firmware tree from the
working directory, but `userspace-compile` resolves it from `user.qmk_home` —
leave that pointing at mainline and you get
`ValueError: Invalid keyboard: zsa/moonlander/reva`, because mainline has no
revision subfolders.

QMK resolves a symlinked userspace to its real path — `userspace-doctor` will
report the dotfiles location, not the `~/.config` symlink. Compare resolved
paths, not literal ones.

**mappy must not configure QMK this way.** `user.qmk_home` is a single global
setting, but firmware trees are not interchangeable — a ZSA board needs ZSA's
fork, another board needs mainline, a third needs its own vendor fork. A tool
that rewrites a global config per keyboard breaks the moment you own two
keyboards. mappy passes `QMK_HOME` as an environment variable per invocation
and never touches `qmk config`.

### 6. Compile and flash

```sh
qmk flash -kb zsa/moonlander/reva -km vimster
```

With the userspace configured this works from any directory. `qmk
userspace-compile` builds every entry in `build_targets` but does not flash.

Run the command **first**, then press the reset button (pinhole on the left
half, near the USB-C port) when it prints `Waiting for bootloader...`. The
board stops being a keyboard the moment it enters DFU mode, so you cannot type
the command afterwards. Unplugging and replugging returns it to normal
firmware if you get stuck.

`dfu-util` reports `dfuERROR, status(10) = Device's firmware is corrupt`
before flashing. This is benign — it clears the status and proceeds.

## Board revision

`reva` and `revb` are separate keyboard targets and are **not**
interchangeable:

| | rev A | rev B |
|---|---|---|
| USB PID | `0x1969` | `0x1972` |
| EEPROM | `i2c` (physical chip) | `wear_leveling` / `embedded_flash` |
| Bootloader | `stm32-dfu` | `custom` |
| Flash address | `0x08000000` | `0x08002000` |

**Detect it from USB rather than asking the user.** The PID is set by the
firmware descriptor, and ZSA's own tooling flashes the revision matching the
hardware, so whatever is currently on the board reports the right answer:

```sh
lsusb -d 3297: # 3297:1969 = reva, 3297:1972 = revb
```

Cross-flashing rev B firmware onto rev A hardware boots and types, but writes
persistent settings to emulated flash while ignoring the physical EEPROM. Not
damaging, but wrong. Recover by flashing the correct revision.

Nothing here can brick the board: the STM32 DFU bootloader is in ROM and the
physical reset button always works.

## Data model

`keymap.c` is the source of truth. JSON is a derived view, never persisted.

The `keymap.json` in an Oryx export is not the layout — it is a six-line module
declaration:

```json
{ "modules": ["zsa/oryx", "zsa/defaults"] }
```

The keycodes are C: three `LAYOUT_moonlander(...)` blocks. Oryx generates them
mechanically, one keycode per position, so they parse easily — but they are
still C, surrounded by code that matters.

### Reading: `qmk c2json` works

```sh
qmk c2json -kb zsa/moonlander/reva -km vimster <path-to-keymap.c> --no-cpp
```

Emits `{keyboard, keymap, layout, layers}` with `layers` as a 3 x 72 array of
keycode strings. Mod-taps (`MT(MOD_LGUI, KC_A)`), `OSL(1)`, `TG(2)` and
`LGUI(KC_X)` all survive intact.

`--no-cpp` matters: it leaves `DUAL_FUNC_0` as a symbol instead of expanding it
to `LT(3, KC_F5)`, preserving the author's name. The `#define` stays in
`keymap.c` where nothing touches it.

### Writing: `qmk json2c` does NOT work

It regenerates only the `keymaps[]` array and discards everything else.
Measured on the Vimster layout: **169 lines in, 26 lines out**. Lost:

- `#define DUAL_FUNC_0 LT(3, KC_F5)` — and the JSON *references*
  `DUAL_FUNC_0`, so the regenerated file does not compile
- `enum custom_keycodes { RGB_SLD = ZSA_SAFE_RANGE }`
- `chordal_hold_layout` (home row mod tuning)
- ~100 lines of RGB matrix code
- `#include "version.h"`

### Therefore: splice, do not regenerate

Locate `const uint16_t PROGMEM keymaps[][MATRIX_ROWS][MATRIX_COLS] = {`, brace
count to its matching `};`, and replace only that byte range. Every other byte
is copied verbatim, so custom code survives by never being parsed.

`ponytail:` brace counting is fooled by braces inside string literals or
comments within the array. Oryx never emits those. Upgrade to a real scanner
only if a hand-written keymap ever hits it.

The correctness test is semantic, not textual — output formatting will not
match Oryx's column alignment and chasing that is wasted effort. Splice the
layers back *unmodified*, re-run `c2json`, and compare the JSON. Then compile.

## Physical geometry

```sh
qmk info -kb zsa/moonlander/reva -f json
```

`layouts.LAYOUT.layout[]` is 72 entries of `{matrix, x, y, w}`, index-aligned
with the 72 keycodes from `c2json`. Index *i* in a layer is index *i* in the
geometry array — that is the whole renderer input.

Fractional `y` values (0.375, 0.125, 0) are the Moonlander's column stagger, so
the real splayed shape is available, not an approximation. `LAYOUT_moonlander`
is an alias for `LAYOUT` (see `layout_aliases`).

None of this is Moonlander-specific. Every keyboard in QMK answers the same
query, which is what makes multi-keyboard support nearly free.

## Decisions

- **Keys are addressed by index** for now. `mappy set 0 42 KC_X` is unusable by
  hand, deliberately: it keeps the CLI an honest machine interface and pushes
  ergonomics into the UI where they belong.
- **External userspace**, so the layout is version-controlled independently of
  the firmware tree.
- **Go** for the CLI. Single static binary, stdlib only — `os/exec`,
  `encoding/json`, `bytes`. The plugin end is QML/JavaScript regardless.

## Layout

```
cli/     the tool: keymap data model, compile, flash
plugin/  Omarchy 4 shell overlay (kinds: ["overlay"]), not started
```
