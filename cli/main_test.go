package main

import (
	"bytes"
	"errors"
	"testing"
)

var testKeymap = keymapJSON{
	Keyboard: "zsa/moonlander/reva",
	Keymap:   "vimster",
	Layout:   "LAYOUT_moonlander",
	Layers: [][]string{
		{"KC_ESCAPE", "MT(MOD_LGUI, KC_A)", "DUAL_FUNC_0"},
		{"KC_TRANSPARENT", "KC_LCBR", "OSL(1)"},
	},
}

func TestPrintKeymap(t *testing.T) {
	tests := []struct {
		name string
		args []string

		want    string
		wantErr error
	}{
		{
			name: "no arguments prints every layer",
			args: nil,
			want: "[0]\n" +
				"   0 KC_ESCAPE\n" +
				"   1 MT(MOD_LGUI, KC_A)\n" +
				"   2 DUAL_FUNC_0\n" +
				"[1]\n" +
				"   0 KC_TRANSPARENT\n" +
				"   1 KC_LCBR\n" +
				"   2 OSL(1)\n",
		},
		{
			name: "a layer prints only that layer",
			args: []string{"1"},
			want: "   0 KC_TRANSPARENT\n" +
				"   1 KC_LCBR\n" +
				"   2 OSL(1)\n",
		},
		{
			// the form worth scripting against: no index, no padding
			name: "a layer and an index print the bare keycode",
			args: []string{"0", "1"},
			want: "MT(MOD_LGUI, KC_A)\n",
		},
		{
			name: "the last layer and index are in range",
			args: []string{"1", "2"},
			want: "OSL(1)\n",
		},
		{
			name:    "layer past the end",
			args:    []string{"2"},
			wantErr: errLayerRange,
		},
		{
			name:    "negative layer",
			args:    []string{"-1"},
			wantErr: errLayerRange,
		},
		{
			name:    "index past the end",
			args:    []string{"0", "3"},
			wantErr: errIndexRange,
		},
		{
			name:    "layer is not a number",
			args:    []string{"x"},
			wantErr: errNotANumber,
		},
		{
			name:    "index is not a number",
			args:    []string{"0", "x"},
			wantErr: errNotANumber,
		},
		{
			name:    "more arguments than a layer and an index",
			args:    []string{"0", "1", "2"},
			wantErr: errTooManyArgs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got bytes.Buffer

			err := printKeymap(&got, testKeymap, tt.args)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("printKeymap() error = %v, expected %v", err, tt.wantErr)
			}

			if got.String() != tt.want {
				t.Fatalf("printKeymap() wrote %q, expected %q", got.String(), tt.want)
			}
		})
	}
}

// The ranges belong in the message: an out of range index is the mistake a
// user actually makes, and "0-71" is the whole answer they need.
func TestPrintKeymapErrorsNameTheRange(t *testing.T) {
	tests := []struct {
		name string
		args []string

		want string
	}{
		{
			name: "layer range",
			args: []string{"9"},
			want: "layer out of range: layer 9, keymap has 2 layers (0-1)",
		},
		{
			name: "index range",
			args: []string{"0", "9"},
			want: "index out of range: index 9, layer has 3 keycodes (0-2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := printKeymap(&bytes.Buffer{}, testKeymap, tt.args)

			if err == nil || err.Error() != tt.want {
				t.Fatalf("printKeymap() error = %v, expected %q", err, tt.want)
			}
		})
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name string
		args []string

		wantOut string
		wantErr error
	}{
		{
			name:    "no command",
			args:    nil,
			wantErr: errNoCommand,
		},
		{
			name:    "unknown command",
			args:    []string{"nope"},
			wantErr: errUnknownCommand,
		},
		{
			name:    "help",
			args:    []string{"help"},
			wantOut: usage + "\n",
		},
		{
			name:    "-h is the same as help",
			args:    []string{"-h"},
			wantOut: usage + "\n",
		},
		{
			name:    "--help is the same as help",
			args:    []string{"--help"},
			wantOut: usage + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer

			err := run(tt.args, &stdout)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("run() error = %v, expected %v", err, tt.wantErr)
			}

			if stdout.String() != tt.wantOut {
				t.Fatalf("run() wrote %q, expected %q", stdout.String(), tt.wantOut)
			}
		})
	}
}
