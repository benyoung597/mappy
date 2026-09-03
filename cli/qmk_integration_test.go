//go:build integration

// Exercises the real qmk c2json. Needs qmk on PATH and the ZSA fork reachable,
// so it is kept out of `make test`:
//
//	go test -tags integration ./...
package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	fixtureKeyboard = "zsa/moonlander/reva"
	fixtureName     = "vimster"
)

func TestReadKeymap(t *testing.T) {
	tests := []struct {
		name string

		qmkHome  string
		keyboard string
		path     string

		wantErr   error
		wantInErr string
	}{
		{
			name:     "reads the fixture",
			keyboard: fixtureKeyboard,
			path:     fixtureKeymap,
		},
		{
			// filepath.Abs is what makes this work: qmkHome moves the
			// subprocess elsewhere, so a relative path would be read
			// against the firmware tree instead of this directory
			name:     "relative path survives a working directory elsewhere",
			qmkHome:  t.TempDir(),
			keyboard: fixtureKeyboard,
			path:     fixtureKeymap,
		},
		{
			name:     "unknown keyboard",
			keyboard: "nope/nope",
			path:     fixtureKeymap,
			wantErr:  errC2JSON,
			// the reason only ever reaches us through stderr
			wantInErr: "invalid keyboard_folder value",
		},
		{
			name:      "missing keymap file",
			keyboard:  fixtureKeyboard,
			path:      "testdata/does-not-exist.c",
			wantErr:   errC2JSON,
			wantInErr: "No such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keymap, err := readKeymap(tt.qmkHome, tt.keyboard, fixtureName, tt.path)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("readKeymap() error = %v, expected %v", err, tt.wantErr)
			}

			if tt.wantInErr != "" && !strings.Contains(err.Error(), tt.wantInErr) {
				t.Fatalf("readKeymap() error = %v, expected it to mention %q", err, tt.wantInErr)
			}

			if tt.wantErr != nil {
				return
			}

			if keymap.Layout != "LAYOUT_moonlander" {
				t.Fatalf("layout = %q, expected LAYOUT_moonlander", keymap.Layout)
			}

			if len(keymap.Layers) != 3 || len(keymap.Layers[0]) != 72 {
				t.Fatalf("layers = %d x %d, expected 3 x 72", len(keymap.Layers), len(keymap.Layers[0]))
			}
		})
	}
}

// The point of the whole pipeline: read a keymap, change one key, write it back
// with everything else in the file intact.
func TestReadEditSpliceRoundTrip(t *testing.T) {
	keymap, err := readKeymap("", fixtureKeyboard, fixtureName, fixtureKeymap)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	keymap.Layers[0][0] = "KC_GRAVE"

	src, err := os.ReadFile(fixtureKeymap)
	if err != nil {
		t.Fatalf("failed to read the fixture: %v", err)
	}

	out, err := spliceKeymaps(src, renderKeymaps(keymap.Layout, keymap.Layers))
	if err != nil {
		t.Fatalf("splice failed: %v", err)
	}

	edited := filepath.Join(t.TempDir(), "keymap.c")
	if err := os.WriteFile(edited, out, 0o644); err != nil {
		t.Fatalf("failed to write the spliced keymap: %v", err)
	}

	// c2json must accept what we wrote, and report the edit and nothing else
	reread, err := readKeymap("", fixtureKeyboard, fixtureName, edited)
	if err != nil {
		t.Fatalf("c2json rejected the spliced keymap: %v", err)
	}

	if reread.Layers[0][0] != "KC_GRAVE" {
		t.Fatalf("layers[0][0] = %q, expected KC_GRAVE", reread.Layers[0][0])
	}

	keymap.Layers[0][0] = "KC_ESCAPE"
	reread.Layers[0][0] = "KC_ESCAPE"

	if !reflect.DeepEqual(keymap.Layers, reread.Layers) {
		t.Fatalf("layers changed beyond the edited key")
	}
}
