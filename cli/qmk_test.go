package main

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

const fixtureKeymapJSON = "testdata/keymap.json"

func TestParseKeymapJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string

		want    keymapJSON
		wantErr error
	}{
		{
			name: "every field is read",
			raw:  `{"keyboard":"zsa/moonlander/reva","keymap":"vimster","layout":"LAYOUT_moonlander","layers":[["KC_A"]]}`,
			want: keymapJSON{
				Keyboard: "zsa/moonlander/reva",
				Keymap:   "vimster",
				Layout:   "LAYOUT_moonlander",
				Layers:   [][]string{{"KC_A"}},
			},
		},
		{
			name: "keycodes are preserved verbatim",
			raw:  `{"layout":"LAYOUT_x","layers":[["MT(MOD_LGUI, KC_A)","DUAL_FUNC_0","OSL(1)"]]}`,
			want: keymapJSON{
				Layout: "LAYOUT_x",
				Layers: [][]string{{"MT(MOD_LGUI, KC_A)", "DUAL_FUNC_0", "OSL(1)"}},
			},
		},
		{
			name: "layers of equal length are accepted",
			raw:  `{"layout":"L","layers":[["KC_A","KC_B"],["KC_C","KC_D"],["KC_E","KC_F"]]}`,
			want: keymapJSON{
				Layout: "L",
				Layers: [][]string{{"KC_A", "KC_B"}, {"KC_C", "KC_D"}, {"KC_E", "KC_F"}},
			},
		},
		{
			name:    "malformed json",
			raw:     `{"layout":`,
			wantErr: errParsingKeyMap,
		},
		{
			name:    "layers of the wrong type",
			raw:     `{"layout":"L","layers":"not an array"}`,
			wantErr: errParsingKeyMap,
		},
		{
			// c2json cannot emit this, but a nil pointer here used to panic
			name:    "json null",
			raw:     `null`,
			wantErr: errNoLayout,
		},
		{
			name:    "no layout field",
			raw:     `{"layers":[["KC_A"]]}`,
			wantErr: errNoLayout,
		},
		{
			name:    "no layers field",
			raw:     `{"layout":"L"}`,
			wantErr: errNoLayers,
		},
		{
			name:    "empty layers array",
			raw:     `{"layout":"L","layers":[]}`,
			wantErr: errNoLayers,
		},
		{
			name:    "single layer holding no keycodes",
			raw:     `{"layout":"L","layers":[[]]}`,
			wantErr: errNoLayers,
		},
		{
			// what a truncated keymap.c looks like: c2json exits 0 and says nothing
			name:    "second layer is short",
			raw:     `{"layout":"L","layers":[["KC_A","KC_B"],["KC_C"]]}`,
			wantErr: errRaggedLayers,
		},
		{
			name:    "a later layer is long",
			raw:     `{"layout":"L","layers":[["KC_A"],["KC_B"],["KC_C","KC_D"]]}`,
			wantErr: errRaggedLayers,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseKeymapJSON([]byte(tt.raw))

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("parseKeymapJSON() error = %v, expected %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseKeymapJSON() = %+v, expected %+v", got, tt.want)
			}
		})
	}
}

// Guards against c2json's output shape drifting under us.
func TestParseKeymapJSONReadsRealOutput(t *testing.T) {
	raw, err := os.ReadFile(fixtureKeymapJSON)
	if err != nil {
		t.Fatalf("failed to read the fixture keymap json: %v", err)
	}

	keymap, err := parseKeymapJSON(raw)
	if err != nil {
		t.Fatalf("real c2json output was rejected: %v", err)
	}

	if keymap.Layout != "LAYOUT_moonlander" {
		t.Fatalf("layout = %q, expected LAYOUT_moonlander", keymap.Layout)
	}

	if len(keymap.Layers) != 3 {
		t.Fatalf("layers = %d, expected 3", len(keymap.Layers))
	}

	for i, layer := range keymap.Layers {
		if len(layer) != 72 {
			t.Fatalf("layer %d has %d keycodes, expected 72", i, len(layer))
		}
	}

	// --no-cpp leaves the author's #define alone instead of expanding it
	if keymap.Layers[0][40] != "DUAL_FUNC_0" {
		t.Fatalf("layers[0][40] = %q, expected DUAL_FUNC_0", keymap.Layers[0][40])
	}
}
