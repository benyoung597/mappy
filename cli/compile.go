package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

var (
	errCompile        = errors.New("qmk compile failed")
	errNoKeymapName   = errors.New("cannot tell which keymap to build")
	errKeymapNotInDir = errors.New("keymap.c is not inside a keymaps directory")
)

func compile(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("compile", flag.ContinueOnError)
	flags.SetOutput(stdout)

	qmkHome := flags.String("qmk-home", "", "firmware tree to run qmk in")
	keyboard := flags.String("keyboard", "", "keyboard to build for; detected from USB when unset")
	name := flags.String("keymap", "", "keymap to build; taken from -file when unset")
	path := flags.String("file", "", "keymap.c whose keymap name should be built")
	clean := flags.Bool("clean", false, "remove object files before compiling")
	parallel := flags.Int("j", 0, "parallel make jobs, 0 for unlimited")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return err
	}

	if flags.NArg() != 0 {
		return fmt.Errorf("%w: compile takes no arguments", errTooManyArgs)
	}

	board, err := resolveKeyboard(*keyboard)
	if err != nil {
		return err
	}

	keymap, err := resolveKeymapName(*name, *path)
	if err != nil {
		return err
	}

	return runCompile(*qmkHome, board, keymap, *clean, *parallel, stdout)
}

// qmk resolves the keymap from the userspace by name, so the keymap name is
// what connects an edit to a build. QMK's own convention is that the name is
// the directory the keymap.c sits in, which is what makes
//
//	mappy set -file X ... && mappy compile -file X
//
// build the file that was just edited. A path outside a keymaps directory
// gives a name qmk cannot find, so the mistake is loud rather than silently
// building a different keymap.
func resolveKeymapName(name, path string) (string, error) {
	if name != "" {
		return name, nil
	}

	if path == "" {
		return "", fmt.Errorf("%w: pass -keymap or -file", errNoKeymapName)
	}

	return keymapNameFromPath(path)
}

func keymapNameFromPath(path string) (string, error) {
	dir := filepath.Dir(filepath.Clean(path))

	name := filepath.Base(dir)
	parent := filepath.Base(filepath.Dir(dir))

	if parent != "keymaps" || name == "." || name == string(filepath.Separator) {
		return "", fmt.Errorf("%w: %s", errKeymapNotInDir, path)
	}

	return name, nil
}

// Streams qmk's output rather than capturing it. A build is slow enough that
// watching it matters, and the progress lines are the whole point.
func runCompile(qmkHome, keyboard, keymap string, clean bool, parallel int, stdout io.Writer) error {
	args := []string{"compile", "-kb", keyboard, "-km", keymap}

	if clean {
		args = append(args, "--clean")
	}

	if parallel != 0 {
		args = append(args, "-j", strconv.Itoa(parallel))
	}

	cmd := exec.Command("qmk", args...)
	cmd.Dir = qmkHome
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s %s: %w", errCompile, keyboard, keymap, err)
	}

	return nil
}
