package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

var (
	errNoCommand      = errors.New("no command given")
	errUnknownCommand = errors.New("unknown command")
	errTooManyArgs    = errors.New("too many arguments")
	errLayerRange     = errors.New("layer out of range")
	errIndexRange     = errors.New("index out of range")
	errNotANumber     = errors.New("not a number")
)

const usage = `usage: mappy <command> [flags] [args]

commands:
  get [layer [index]]   print the keycodes in a keymap.c
  help                  print this message

run "mappy <command> -h" for the flags a command takes`

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mappy:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%w\n\n%s", errNoCommand, usage)
	}

	switch args[0] {
	case "get":
		return get(args[1:], stdout)
	case "help", "-h", "--help":
		_, err := fmt.Fprintln(stdout, usage)

		return err
	default:
		return fmt.Errorf("%w %q\n\n%s", errUnknownCommand, args[0], usage)
	}
}

func get(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("get", flag.ContinueOnError)
	flags.SetOutput(stdout)

	// Empty lets qmk resolve the tree itself, from the working directory and
	// then user.qmk_home. Only worth setting when more than one tree exists.
	qmkHome := flags.String("qmk-home", "", "firmware tree to run qmk in")
	keyboard := flags.String("keyboard", "", "keyboard to read the keymap as, e.g. zsa/moonlander/reva")
	// c2json does not validate this, it copies it into its "keymap" field.
	name := flags.String("keymap", "default", "keymap name to report")
	path := flags.String("file", "", "path to the keymap.c to read")

	if err := flags.Parse(args); err != nil {
		// -h is a request, not a failure: Parse has already printed the usage
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

	keymap, err := readKeymap(*qmkHome, *keyboard, *name, *path)
	if err != nil {
		return err
	}

	return printKeymap(stdout, keymap, flags.Args())
}

// Prints every layer, one layer, or a single keycode, depending on how many
// positional arguments are given. The bare single-keycode form is the one
// worth scripting against.
func printKeymap(w io.Writer, keymap keymapJSON, args []string) error {
	if len(args) > 2 {
		return fmt.Errorf("%w: get takes at most a layer and an index", errTooManyArgs)
	}

	if len(args) == 0 {
		for i := range keymap.Layers {
			if _, err := fmt.Fprintf(w, "[%d]\n", i); err != nil {
				return err
			}

			if err := printLayer(w, keymap.Layers[i]); err != nil {
				return err
			}
		}

		return nil
	}

	layer, err := parseLayer(keymap, args[0])
	if err != nil {
		return err
	}

	if len(args) == 1 {
		return printLayer(w, keymap.Layers[layer])
	}

	index, err := parseIndex(keymap.Layers[layer], args[1])
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, keymap.Layers[layer][index])

	return err
}

func printLayer(w io.Writer, layer []string) error {
	for i, keycode := range layer {
		if _, err := fmt.Fprintf(w, "  %2d %s\n", i, keycode); err != nil {
			return err
		}
	}

	return nil
}

func parseLayer(keymap keymapJSON, arg string) (int, error) {
	layer, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("%w: layer %q", errNotANumber, arg)
	}

	if layer < 0 || layer >= len(keymap.Layers) {
		return 0, fmt.Errorf("%w: layer %d, keymap has %d layers (0-%d)",
			errLayerRange, layer, len(keymap.Layers), len(keymap.Layers)-1)
	}

	return layer, nil
}

func parseIndex(layer []string, arg string) (int, error) {
	index, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("%w: index %q", errNotANumber, arg)
	}

	if index < 0 || index >= len(layer) {
		return 0, fmt.Errorf("%w: index %d, layer has %d keycodes (0-%d)",
			errIndexRange, index, len(layer), len(layer)-1)
	}

	return index, nil
}
