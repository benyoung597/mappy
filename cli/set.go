package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

var (
	errWrongArgCount   = errors.New("set takes a layer, an index and a keycode")
	errEmptyKeycode    = errors.New("keycode is empty")
	errKeycodeNewline  = errors.New("keycode contains a newline")
	errKeycodeParens   = errors.New("keycode has unbalanced parentheses")
	errVerifyKeycode   = errors.New("written keymap does not hold the new keycode")
	errVerifyUnrelated = errors.New("written keymap changed more than the one keycode")
)

func set(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("set", flag.ContinueOnError)
	flags.SetOutput(stdout)

	qmkHome := flags.String("qmk-home", "", "firmware tree to run qmk in")
	keyboard := flags.String("keyboard", "", "keyboard to read the keymap as, e.g. zsa/moonlander/reva")
	name := flags.String("keymap", "default", "keymap name to report")
	path := flags.String("file", "", "path to the keymap.c to edit")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return err
	}

	if *keyboard == "" {
		return errors.New("-keyboard is required")
	}

	if *path == "" {
		return errors.New("-file is required")
	}

	if flags.NArg() != 3 {
		return fmt.Errorf("%w, got %d arguments", errWrongArgCount, flags.NArg())
	}

	keycode := flags.Arg(2)

	if err := validateKeycode(keycode); err != nil {
		return err
	}

	// A keymap.c reached through a stowed symlink must be written where it
	// really lives, or the rename replaces the link with a regular file.
	target, err := filepath.EvalSymlinks(*path)
	if err != nil {
		return err
	}

	keymap, err := readKeymap(*qmkHome, *keyboard, *name, target)
	if err != nil {
		return err
	}

	layer, err := parseLayer(keymap, flags.Arg(0))
	if err != nil {
		return err
	}

	index, err := parseIndex(keymap.Layers[layer], flags.Arg(1))
	if err != nil {
		return err
	}

	was := keymap.Layers[layer][index]

	if was == keycode {
		return nil
	}

	before := cloneLayers(keymap.Layers)

	keymap.Layers[layer][index] = keycode

	if err := writeKeymap(*qmkHome, *keyboard, *name, target, keymap, before, layer, index); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "%d %d: %s -> %s\n", layer, index, was, keycode)

	return err
}

// Splices the edited layers into the file, then reads the result back through
// c2json before it replaces the original. A splice that produces a keymap.c
// qmk cannot parse never reaches the real file.
func writeKeymap(qmkHome, keyboard, name, target string, keymap keymapJSON, before [][]string, layer, index int) error {
	src, err := os.ReadFile(target)
	if err != nil {
		return err
	}

	out, err := spliceKeymaps(src, renderKeymaps(keymap.Layout, keymap.Layers))
	if err != nil {
		return err
	}

	info, err := os.Stat(target)
	if err != nil {
		return err
	}

	// Same directory, so the rename below cannot cross a filesystem.
	temp, err := os.CreateTemp(filepath.Dir(target), filepath.Base(target)+".mappy-*")
	if err != nil {
		return err
	}

	defer os.Remove(temp.Name())

	if _, err := temp.Write(out); err != nil {
		temp.Close()

		return err
	}

	if err := temp.Close(); err != nil {
		return err
	}

	// CreateTemp opens 0600, which is not what the keymap had.
	if err := os.Chmod(temp.Name(), info.Mode().Perm()); err != nil {
		return err
	}

	if err := verifyKeymap(qmkHome, keyboard, name, temp.Name(), before, layer, index, keymap.Layers[layer][index]); err != nil {
		return err
	}

	return os.Rename(temp.Name(), target)
}

func verifyKeymap(qmkHome, keyboard, name, temp string, before [][]string, layer, index int, keycode string) error {
	written, err := readKeymap(qmkHome, keyboard, name, temp)
	if err != nil {
		return err
	}

	if got := written.Layers[layer][index]; got != keycode {
		return fmt.Errorf("%w: layer %d index %d is %q, expected %q", errVerifyKeycode, layer, index, got, keycode)
	}

	// Compare against the layers as they were read, with only the edit applied,
	// so anything else the splice disturbed shows up here.
	expected := cloneLayers(before)
	expected[layer][index] = keycode

	if !reflect.DeepEqual(written.Layers, expected) {
		return errVerifyUnrelated
	}

	return nil
}

// Rejects the keycodes that would corrupt the array rather than merely fail to
// compile. A misspelt but well formed keycode is the compiler's problem: no
// keycode list can be authoritative when layouts define their own.
func validateKeycode(keycode string) error {
	if strings.TrimSpace(keycode) == "" {
		return errEmptyKeycode
	}

	if strings.ContainsAny(keycode, "\n\r") {
		return errKeycodeNewline
	}

	depth := 0

	for _, r := range keycode {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		}

		if depth < 0 {
			return fmt.Errorf("%w: %q", errKeycodeParens, keycode)
		}
	}

	if depth != 0 {
		return fmt.Errorf("%w: %q", errKeycodeParens, keycode)
	}

	return nil
}

func cloneLayers(layers [][]string) [][]string {
	clone := make([][]string, len(layers))

	for i, layer := range layers {
		clone[i] = append([]string(nil), layer...)
	}

	return clone
}
