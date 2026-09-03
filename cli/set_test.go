package main

import (
	"errors"
	"testing"
)

func TestValidateKeycode(t *testing.T) {
	tests := []struct {
		name    string
		keycode string

		wantErr error
	}{
		{
			name:    "a plain keycode",
			keycode: "KC_ESCAPE",
		},
		{
			name:    "a macro call with an argument",
			keycode: "OSL(1)",
		},
		{
			// the comma and space are why keycodes stay opaque strings
			name:    "a mod tap",
			keycode: "MT(MOD_LGUI, KC_A)",
		},
		{
			name:    "nested calls",
			keycode: "LT(3, LGUI(KC_X))",
		},
		{
			// a layout's own #define, which no keycode list would know
			name:    "a defined symbol",
			keycode: "DUAL_FUNC_0",
		},
		{
			name:    "empty",
			keycode: "",
			wantErr: errEmptyKeycode,
		},
		{
			name:    "only whitespace",
			keycode: "   ",
			wantErr: errEmptyKeycode,
		},
		{
			name:    "a newline would split the argument list",
			keycode: "KC_A\nKC_B",
			wantErr: errKeycodeNewline,
		},
		{
			// the realistic corruption: a shell quoting slip
			name:    "unclosed paren",
			keycode: "MT(MOD_LGUI, KC_A",
			wantErr: errKeycodeParens,
		},
		{
			name:    "unopened paren",
			keycode: "MOD_LGUI, KC_A)",
			wantErr: errKeycodeParens,
		},
		{
			name:    "closed before opened",
			keycode: ")KC_A(",
			wantErr: errKeycodeParens,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKeycode(tt.keycode)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validateKeycode(%q) = %v, expected %v", tt.keycode, err, tt.wantErr)
			}
		})
	}
}

func TestCloneLayers(t *testing.T) {
	original := [][]string{{"KC_A", "KC_B"}, {"KC_C"}}

	clone := cloneLayers(original)

	clone[0][0] = "KC_X"

	if original[0][0] != "KC_A" {
		t.Fatalf("writing to the clone changed the original: %v", original)
	}
}
