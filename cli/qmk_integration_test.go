//go:build integration

// Exercises the real qmk c2json. Needs qmk on PATH and the ZSA fork reachable,
// so it is kept out of `make test`:
//
//	go test -tags integration ./...
package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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

// get is the whole reader path with a terminal on the end of it.
func TestGetCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string

		want    string
		wantErr string
	}{
		{
			name: "a single keycode",
			args: []string{"0", "40"},
			want: "DUAL_FUNC_0\n",
		},
		{
			// -keyboard is optional now, detected from USB, but the file is not
			name:    "file is required",
			args:    nil,
			wantErr: "-file is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"get"}

			if tt.wantErr == "" {
				args = append(args, "-keyboard", fixtureKeyboard, "-keymap", fixtureName, "-file", fixtureKeymap)
			}

			args = append(args, tt.args...)

			var stdout bytes.Buffer

			err := run(args, &stdout)

			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("run() error = %v, expected %q", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("run() error = %v", err)
			}

			if stdout.String() != tt.want {
				t.Fatalf("run() wrote %q, expected %q", stdout.String(), tt.want)
			}
		})
	}
}

func TestSetCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string

		want    string
		wantErr error
	}{
		{
			name: "replaces one keycode",
			args: []string{"0", "0", "KC_GRAVE"},
			want: "0 0: KC_ESCAPE -> KC_GRAVE\n",
		},
		{
			name: "a keycode holding a comma",
			args: []string{"1", "0", "MT(MOD_LGUI, KC_A)"},
			want: "1 0: KC_TRANSPARENT -> MT(MOD_LGUI, KC_A)\n",
		},
		{
			// nothing to do, so nothing is written and nothing is said
			name: "setting the keycode already there",
			args: []string{"0", "40", "DUAL_FUNC_0"},
			want: "",
		},
		{
			name:    "layer out of range",
			args:    []string{"9", "0", "KC_A"},
			wantErr: errLayerRange,
		},
		{
			name:    "index out of range",
			args:    []string{"0", "99", "KC_A"},
			wantErr: errIndexRange,
		},
		{
			name:    "unbalanced parens are refused before anything is read",
			args:    []string{"0", "0", "MT(MOD_LGUI, KC_A"},
			wantErr: errKeycodeParens,
		},
		{
			name:    "too few arguments",
			args:    []string{"0", "0"},
			wantErr: errWrongArgCount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tempKeymap(t)

			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read the temp keymap: %v", err)
			}

			args := append([]string{"set", "-keyboard", fixtureKeyboard, "-keymap", fixtureName, "-file", path}, tt.args...)

			var stdout bytes.Buffer

			err = run(args, &stdout)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("run() error = %v, expected %v", err, tt.wantErr)
			}

			if stdout.String() != tt.want {
				t.Fatalf("run() wrote %q, expected %q", stdout.String(), tt.want)
			}

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to re-read the temp keymap: %v", err)
			}

			// Anything that did not print a change must not have touched the file
			if tt.want == "" && !bytes.Equal(before, after) {
				t.Fatalf("the keymap was modified despite no change being reported")
			}

			if tt.wantErr != nil || tt.want == "" {
				return
			}

			// The edit is readable back, and the custom code is still there
			keymap, err := readKeymap("", fixtureKeyboard, fixtureName, path)
			if err != nil {
				t.Fatalf("the written keymap could not be read back: %v", err)
			}

			layer, index := 0, 0
			if tt.args[0] == "1" {
				layer = 1
			}

			if keymap.Layers[layer][index] != tt.args[2] {
				t.Fatalf("layers[%d][%d] = %q, expected %q", layer, index, keymap.Layers[layer][index], tt.args[2])
			}

			for _, want := range []string{"#define DUAL_FUNC_0", "RGB_SLD", "chordal_hold_layout"} {
				if !bytes.Contains(after, []byte(want)) {
					t.Fatalf("custom code was lost: %s", want)
				}
			}
		})
	}
}

// The file mode survives the temp-file-and-rename dance.
func TestSetPreservesMode(t *testing.T) {
	path := tempKeymap(t)

	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}

	args := []string{"set", "-keyboard", fixtureKeyboard, "-keymap", fixtureName, "-file", path, "0", "0", "KC_GRAVE"}

	if err := run(args, &bytes.Buffer{}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, expected 0640", info.Mode().Perm())
	}
}

// A keymap.c reached through a symlink is written where it really lives.
func TestSetFollowsSymlinks(t *testing.T) {
	real := tempKeymap(t)
	link := filepath.Join(t.TempDir(), "link.c")

	if err := os.Symlink(real, link); err != nil {
		t.Fatalf("symlink failed: %v", err)
	}

	args := []string{"set", "-keyboard", fixtureKeyboard, "-keymap", fixtureName, "-file", link, "0", "0", "KC_GRAVE"}

	if err := run(args, &bytes.Buffer{}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("lstat failed: %v", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("the symlink was replaced with a regular file")
	}

	keymap, err := readKeymap("", fixtureKeyboard, fixtureName, real)
	if err != nil {
		t.Fatalf("the real file could not be read back: %v", err)
	}

	if keymap.Layers[0][0] != "KC_GRAVE" {
		t.Fatalf("layers[0][0] = %q, expected KC_GRAVE", keymap.Layers[0][0])
	}
}

// The point of verifying before renaming: a splice qmk cannot parse must leave
// the original file exactly as it was, and leave no temp file behind either.
func TestSetRollsBackAnUnparseableWrite(t *testing.T) {
	path := tempKeymap(t)

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read the temp keymap: %v", err)
	}

	keymap, err := readKeymap("", fixtureKeyboard, fixtureName, path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	original := cloneLayers(keymap.Layers)

	// A brace closes the array early. validateKeycode only guards parens, so
	// this gets past it and the verify step is the thing that has to catch it.
	keymap.Layers[0][0] = "KC_GRAVE}"

	err = writeKeymap("", fixtureKeyboard, fixtureName, path, keymap, original, 0, 0)
	if err == nil {
		t.Fatalf("an unparseable write was accepted")
	}

	t.Logf("rejected with: %v", err)

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to re-read the temp keymap: %v", err)
	}

	if !bytes.Equal(before, after) {
		t.Fatalf("the original keymap was modified by a failed write")
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("failed to list the directory: %v", err)
	}

	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".mappy-") {
			t.Fatalf("a temp file was left behind: %s", entry.Name())
		}
	}
}

// The reason set edits bytes instead of rewriting the array.
func TestSetProducesAOneLineDiff(t *testing.T) {
	path := tempKeymap(t)

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read the temp keymap: %v", err)
	}

	args := []string{"set", "-keyboard", fixtureKeyboard, "-keymap", fixtureName, "-file", path, "0", "0", "KC_GRAVE"}

	if err := run(args, &bytes.Buffer{}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to re-read the temp keymap: %v", err)
	}

	was := strings.Split(string(before), "\n")
	now := strings.Split(string(after), "\n")

	if len(was) != len(now) {
		t.Fatalf("line count changed: %d -> %d", len(was), len(now))
	}

	changed := 0

	for i := range was {
		if was[i] != now[i] {
			changed++
		}
	}

	if changed != 1 {
		t.Fatalf("%d lines changed, expected 1", changed)
	}

	// and the column padding is still holding the row together
	if len(was[18]) != len(now[18]) {
		t.Fatalf("row width changed: %d -> %d", len(was[18]), len(now[18]))
	}
}
