package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

var (
	errNoAnchor                                   = errors.New("no anchor found in bytes array")
	errNoLayers                                   = errors.New("no layers were provided")
	errOpeningBracketNotFound                     = errors.New("no opening bracket found")
	errClosingBracketNotFound                     = errors.New("no closing bracket found")
	errMissingTerminatingSemicolon                = errors.New("terminating semicolon not found")
	errSemicolonNotImmediatelyAfterClosingBracket = errors.New("terminating semicolon not immediately after terminating bracket")
)

var keymapsAnchor = "PROGMEM keymaps["
var ledAnchor = "PROGMEM ledmap["

// Locates the keymapps[] array. Returns the byte range to replace,
// where end is just past the closing "};".
func findKeymapsSpan(src []byte) (start, end int, err error) {
	if len(src) == 0 {
		return 0, 0, errNoAnchor
	}

	index := bytes.Index(src, []byte(keymapsAnchor))
	if index == -1 {
		return 0, 0, errNoAnchor
	}

	index += len(keymapsAnchor) - 1

	// Loop backwards to find the first occurence of a new line and get the index of the newline
	// plus one. This is the definition of the defined array
	start = 0
	end = 0

	for i := index; i >= 0; i-- {
		if src[i] == '\n' {
			start = i + 1
			break
		}
	}

	depth := 0
	braceIndex := 0

	for i := start; i < len(src); i++ {
		if src[i] == '{' {
			depth = 1
			braceIndex = i + 1
			break
		}
	}

	if braceIndex == 0 {
		return 0, 0, errOpeningBracketNotFound
	}

	for i := braceIndex; i < len(src); i++ {
		if src[i] == '{' {
			depth++
		}

		if src[i] == '}' {
			depth--
		}

		if depth == 0 {
			end = i
			break
		}
	}

	if depth > 0 {
		return 0, 0, errClosingBracketNotFound
	}

	if end+1 >= len(src) {
		return 0, 0, errMissingTerminatingSemicolon
	}

	if src[end+1] != ';' {
		return 0, 0, errSemicolonNotImmediatelyAfterClosingBracket
	}

	end += 2

	return start, end, nil
}

func findLedmapSpan(src []byte) (start, end int, err error) {
	return 0, 0, nil
}

// Emits a replacement array. Layout is the macro name ("LAYOUT_moonlander"),
// which comes from the c2json's "layout" field - don't hardcode it
//
// Layers is [layer][position] - a flat list of keycodes per layer, straight off
// c2json. Not [row][col]: the LAYOUT_ macro expands the flat list into the
// matrix and pads the unwired cells, so 72 keycodes become a [12][7] array.
//
// The leading newline and trailing "};" are ours because spliceKeymaps keeps
// src up to and including the opening brace, and replaces through the ";".
//
//	renderKeymaps("LAYOUT_x", [][]string{{"KC_A", "OSL(1)"}, {"KC_B", "KC_C"}})
//
// returns exactly this, as one byte slice:
//
//	"\n  [0] = LAYOUT_x(\n    KC_A,\n    OSL(1)\n  ),\n  [1] = LAYOUT_x(\n    KC_B,\n    KC_C\n  ),\n};"
//
// Per layer: open with "\n  [i] = <layout>(\n    ", join the keycodes with
// ",\n    " (the comma belongs to the separator, which is what leaves the last
// keycode bare), then close with "\n  ),". After the last layer, "\n};".
//
// A bare last keycode is not cosmetic: LAYOUT_x is a macro call, not an
// initializer, so a trailing comma there will not compile. Spliced, it reads:
//
//	const uint16_t PROGMEM keymaps[][MATRIX_ROWS][MATRIX_COLS] = {
//	  [0] = LAYOUT_x(
//	    KC_A,
//	    OSL(1)
//	  ),
//	  [1] = LAYOUT_x(
//	    KC_B,
//	    KC_C
//	  ),
//	};
func renderKeymaps(layout string, layers [][]string) []byte {
	if len(layers) == 0 {
		return []byte{}
	}

	var b bytes.Buffer

	for index, layer := range layers {
		fmt.Fprintf(&b, "\n  [%d] = %s(\n    %s\n  ),", index, layout, strings.Join(layer, ",\n    "))
	}

	b.WriteString("\n};")

	return b.Bytes()
}

func spliceKeymaps(src, body []byte) ([]byte, error) {
	if len(body) == 0 {
		return nil, errNoLayers
	}

	start, end, err := findKeymapsSpan(src)
	if err != nil {
		return nil, err
	}

	brace := start + bytes.IndexByte(src[start:end], '{')

	out := make([]byte, 0, brace+1+len(body)+len(src)-end)
	out = append(out, src[:brace+1]...)
	out = append(out, body...)
	out = append(out, src[end:]...)

	return out, nil
}
