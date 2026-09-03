package main

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

const fixtureKeymap = "testdata/keymap.c"

func TestFindKeymapsSpan(t *testing.T) {
	tests := []struct {
		name  string
		input []byte

		wantStart int
		wantEnd   int
		wantErr   error
	}{
		{
			name:      "empty byte array provided",
			input:     []byte{},
			wantStart: 0,
			wantEnd:   0,
			wantErr:   errNoAnchor,
		},
		{
			name:      "no PROGMEM keymaps[ found in the bytes provided",
			input:     []byte("these bytes don't include the key phrase that we are looking for"),
			wantStart: 0,
			wantEnd:   0,
			wantErr:   errNoAnchor,
		},
		{
			name:      "opening bracket not found after PROGMEM keymaps[",
			input:     []byte("const uint16_t PROGMEM keymaps[][MATRIX_ROWS][MATRIX_COLS] = "),
			wantStart: 0,
			wantEnd:   0,
			wantErr:   errOpeningBracketNotFound,
		},
		{
			name:      "closing bracket not found",
			input:     []byte("const uint16_t PROGMEM keymaps[][MATRIX_ROWS][MATRIX_COLS] = {"),
			wantStart: 0,
			wantEnd:   0,
			wantErr:   errClosingBracketNotFound,
		},
		{
			name:      "brackets balance but no terminating semicolon",
			input:     []byte("const uint16_t PROGMEM keymaps[][MATRIX_ROWS][MATRIX_COLS] = {}"),
			wantStart: 0,
			wantEnd:   0,
			wantErr:   errMissingTerminatingSemicolon,
		},
		{
			name:      "semicolon belongs to a later statement",
			input:     []byte("const uint16_t PROGMEM keymaps[][1][1] = {{{KC_A}}}\nint other = 1;\n"),
			wantStart: 0,
			wantEnd:   0,
			wantErr:   errSemicolonNotImmediatelyAfterClosingBracket,
		},
		{
			name:      "array terminates at end of file",
			input:     []byte("const uint16_t PROGMEM keymaps[][1][1] = {{{KC_A}}};"),
			wantStart: 0,
			wantEnd:   52,
			wantErr:   nil,
		},
		{
			name:      "successfully find the keymaps span",
			input:     []byte("x\nconst uint16_t PROGMEM keymaps[][1][1] = {{{KC_A}}};\ny"),
			wantStart: 2,
			wantEnd:   54,
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := findKeymapsSpan(tt.input)

			if tt.wantStart != start || tt.wantEnd != end || !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"findKeymapsSpan() = %d, %d, %v, expected %d, %d, %v",
					start, end, err,
					tt.wantStart, tt.wantEnd, tt.wantErr,
				)
			}
		})
	}
}

func TestFindKeymapsSpanHasRealAnchor(t *testing.T) {
	src, err := os.ReadFile(fixtureKeymap)
	if err != nil {
		t.Fatalf("failed to find the fixture key map: %v", err)
	}

	start, end, err := findKeymapsSpan(src)
	if err != nil {
		t.Fatalf("issue finding anchor in provided src: %v", err)
	}

	if end <= start {
		t.Fatalf("end was equal to or less than start: %d <= %d", end, start)
	}

	span := src[start:end]

	if !bytes.HasPrefix(span, []byte("const uint16_t PROGMEM keymaps")) {
		t.Fatalf("src has the incorrect prefix. looking for: const uint16_t PROGMEM keymaps")
	}

	if start < 0 || end > len(src) {
		t.Fatalf("invalid values of start and end returned: %d %d", start, end)
	}

	if !bytes.HasSuffix(span, []byte("};")) {
		t.Fatalf("the semicolon does not immediately follow the closing bracket")
	}
}

func TestRenderKeymaps(t *testing.T) {
	tests := []struct {
		name   string
		layout string
		layers [][]string

		want string
	}{
		{
			name:   "no layers renders nothing",
			layout: "LAYOUT_x",
			layers: nil,
			want:   "",
		},
		{
			name:   "single layer, single keycode takes no comma",
			layout: "LAYOUT_x",
			layers: [][]string{{"KC_A"}},
			want:   "\n  [0] = LAYOUT_x(\n    KC_A\n  ),\n};",
		},
		{
			name:   "layer index counts up and every layer takes a comma",
			layout: "LAYOUT_x",
			layers: [][]string{{"KC_A", "OSL(1)"}, {"KC_B", "KC_C"}},
			want:   "\n  [0] = LAYOUT_x(\n    KC_A,\n    OSL(1)\n  ),\n  [1] = LAYOUT_x(\n    KC_B,\n    KC_C\n  ),\n};",
		},
		{
			name:   "keycodes are opaque strings",
			layout: "LAYOUT_moonlander",
			layers: [][]string{{"MT(MOD_LGUI, KC_A)", "DUAL_FUNC_0", "LGUI(KC_X)"}},
			want:   "\n  [0] = LAYOUT_moonlander(\n    MT(MOD_LGUI, KC_A),\n    DUAL_FUNC_0,\n    LGUI(KC_X)\n  ),\n};",
		},
		{
			name:   "layout name is not hardcoded",
			layout: "LAYOUT",
			layers: [][]string{{"KC_A"}},
			want:   "\n  [0] = LAYOUT(\n    KC_A\n  ),\n};",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderKeymaps(tt.layout, tt.layers)

			if string(got) != tt.want {
				t.Fatalf("renderKeymaps() = %q, expected %q", string(got), tt.want)
			}
		})
	}
}

func TestSpliceKeymaps(t *testing.T) {
	const decl = "const uint16_t PROGMEM keymaps[][1][1] = {"

	tests := []struct {
		name string
		src  string
		body string

		want    string
		wantErr error
	}{
		{
			name:    "empty body is rejected",
			src:     decl + "old};\n",
			body:    "",
			wantErr: errNoLayers,
		},
		{
			name:    "span errors propagate",
			src:     "nothing to anchor on here\n",
			body:    "X};",
			wantErr: errNoAnchor,
		},
		{
			name: "bytes either side of the array are preserved",
			src:  "before\n" + decl + "old};\nafter\n",
			body: "\n  [0] = X(\n    KC_A\n  ),\n};",
			want: "before\n" + decl + "\n  [0] = X(\n    KC_A\n  ),\n};\nafter\n",
		},
		{
			name: "declaration line is preserved verbatim, not regenerated",
			src:  "__attribute__((weak)) const uint16_t PROGMEM keymaps[NUM_LAYERS][12][7] = {old};",
			body: "X};",
			want: "__attribute__((weak)) const uint16_t PROGMEM keymaps[NUM_LAYERS][12][7] = {X};",
		},
		{
			name: "array terminating at end of file",
			src:  decl + "old};",
			body: "X};",
			want: decl + "X};",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := spliceKeymaps([]byte(tt.src), []byte(tt.body))

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("spliceKeymaps() error = %v, expected %v", err, tt.wantErr)
			}

			if string(got) != tt.want {
				t.Fatalf("spliceKeymaps() = %q, expected %q", string(got), tt.want)
			}
		})
	}
}

// The whole point of splicing: everything outside keymaps[] survives untouched.
func TestSpliceKeymapsPreservesCustomCode(t *testing.T) {
	src, err := os.ReadFile(fixtureKeymap)
	if err != nil {
		t.Fatalf("failed to find the fixture key map: %v", err)
	}

	start, end, err := findKeymapsSpan(src)
	if err != nil {
		t.Fatalf("issue finding anchor in provided src: %v", err)
	}

	body := renderKeymaps("LAYOUT_moonlander", [][]string{{"KC_A", "DUAL_FUNC_0"}})

	out, err := spliceKeymaps(src, body)
	if err != nil {
		t.Fatalf("splice failed: %v", err)
	}

	if !bytes.Equal(out[:start], src[:start]) {
		t.Fatalf("bytes before the array were not preserved")
	}

	if !bytes.HasSuffix(out, src[end:]) {
		t.Fatalf("bytes after the array were not preserved")
	}

	for _, want := range []string{"#define DUAL_FUNC_0", "RGB_SLD", "chordal_hold_layout", "version.h"} {
		if !bytes.Contains(out, []byte(want)) {
			t.Fatalf("custom code was lost from the spliced output: %s", want)
		}
	}

	// A span we cannot find again is a file mappy can no longer edit.
	if _, _, err := findKeymapsSpan(out); err != nil {
		t.Fatalf("span is not re-findable in the spliced output: %v", err)
	}

	second, err := spliceKeymaps(out, body)
	if err != nil {
		t.Fatalf("second splice failed: %v", err)
	}

	if !bytes.Equal(out, second) {
		t.Fatalf("splicing the same body twice did not produce identical bytes")
	}
}
