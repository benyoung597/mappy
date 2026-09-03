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

var (
	errLayerSpanNotFound   = errors.New("layer not found in keymaps array")
	errKeycodeSpanNotFound = errors.New("keycode not found in layer")
)

// Locates a single keycode inside the keymaps[] array and returns its byte
// range, so one key can be replaced without rewriting the array. That keeps
// Oryx's column padding, and a one keycode edit to a one line diff.
//
// Layers are the parenthesised groups at paren depth zero - the LAYOUT_ calls.
// Keycodes are their comma separated arguments, also at depth zero, which is
// what keeps MT(MOD_LGUI, KC_A) one argument rather than two.
func findKeycodeSpan(src []byte, layer, index int) (start, end int, err error) {
	spanStart, spanEnd, err := findKeymapsSpan(src)
	if err != nil {
		return 0, 0, err
	}

	array := src[spanStart:spanEnd]

	argsStart, argsEnd, err := findLayerArgs(array, layer)
	if err != nil {
		return 0, 0, err
	}

	tokenStart, tokenEnd, err := findArgument(array[argsStart:argsEnd], index)
	if err != nil {
		return 0, 0, err
	}

	return spanStart + argsStart + tokenStart, spanStart + argsStart + tokenEnd, nil
}

func findLayerArgs(array []byte, layer int) (start, end int, err error) {
	depth := 0
	open := 0
	found := 0

	for i := 0; i < len(array); i++ {
		switch array[i] {
		case '(':
			if depth == 0 {
				open = i + 1
			}

			depth++
		case ')':
			depth--

			if depth == 0 {
				if found == layer {
					return open, i, nil
				}

				found++
			}
		}
	}

	return 0, 0, fmt.Errorf("%w: layer %d, found %d", errLayerSpanNotFound, layer, found)
}

func findArgument(args []byte, index int) (start, end int, err error) {
	depth := 0
	token := 0
	found := 0

	for i := 0; i <= len(args); i++ {
		if i == len(args) || (args[i] == ',' && depth == 0) {
			if found == index {
				return trimSpan(args, token, i)
			}

			found++
			token = i + 1

			continue
		}

		switch args[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
	}

	return 0, 0, fmt.Errorf("%w: index %d, found %d", errKeycodeSpanNotFound, index, found)
}

// Narrows start and end onto the token itself, dropping the whitespace and
// newlines Oryx pads its columns with.
func trimSpan(args []byte, start, end int) (int, int, error) {
	for start < end && isSpace(args[start]) {
		start++
	}

	for end > start && isSpace(args[end-1]) {
		end--
	}

	if start == end {
		return 0, 0, errKeycodeSpanNotFound
	}

	return start, end, nil
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// Replaces the keycode at start:end, absorbing the length change into the run
// of padding that follows it so the rest of the row stays in its column. A
// keycode with no padding after it - last on a line, or last in a layer - just
// moves what follows.
func replaceKeycode(src []byte, start, end int, keycode string) []byte {
	tail := end

	if tail < len(src) && src[tail] == ',' {
		tail++
	}

	spaces := 0

	for tail+spaces < len(src) && src[tail+spaces] == ' ' {
		spaces++
	}

	padding := spaces

	if spaces > 0 {
		padding = spaces - (len(keycode) - (end - start))

		if padding < 1 {
			padding = 1
		}
	}

	out := make([]byte, 0, len(src)+len(keycode))
	out = append(out, src[:start]...)
	out = append(out, keycode...)
	out = append(out, src[end:tail]...)
	out = append(out, strings.Repeat(" ", padding)...)
	out = append(out, src[tail+spaces:]...)

	return out
}
