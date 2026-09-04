package main

import (
	"encoding/json"
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
  get [-json] [layer [index]]   print the keycodes in a keymap.c
  set <layer> <index> <keycode> replace one keycode in a keymap.c
  compile                       build the firmware for a keymap
  flash                         build and write the firmware to the keyboard
  patch                         fix the ZSA tree so a modern GCC can build it
  help                          print this message

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
	case "set":
		return set(args[1:], stdout)
	case "compile":
		return compile(args[1:], stdout)
	case "flash":
		return flashCmd(args[1:], stdout)
	case "patch":
		return patch(args[1:], stdout)
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
	keyboard := flags.String("keyboard", "", "keyboard to read the keymap as; detected from USB when unset")
	// c2json does not validate this, it copies it into its "keymap" field.
	name := flags.String("keymap", "default", "keymap name to report")
	path := flags.String("file", "", "path to the keymap.c to read")
	asJSON := flags.Bool("json", false, "print JSON rather than text")

	if err := flags.Parse(args); err != nil {
		// -h is a request, not a failure: Parse has already printed the usage
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return err
	}

	if *path == "" {
		return errors.New("-file is required")
	}

	target, err := resolveKeyboard(*keyboard)
	if err != nil {
		return err
	}

	keymap, err := readKeymap(*qmkHome, target, *name, *path)
	if err != nil {
		return err
	}

	return printKeymap(stdout, keymap, flags.Args(), *asJSON)
}

// Prints every layer, one layer, or a single keycode, depending on how many
// positional arguments are given. The bare single-keycode form is the one
// worth scripting against; -json is the same selection, encoded for jq.
func printKeymap(w io.Writer, keymap keymapJSON, args []string, asJSON bool) error {
	selected, err := selectKeymap(keymap, args)
	if err != nil {
		return err
	}

	if asJSON {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")

		return encoder.Encode(selected)
	}

	switch value := selected.(type) {
	case keymapJSON:
		for i := range value.Layers {
			if _, err := fmt.Fprintf(w, "[%d]\n", i); err != nil {
				return err
			}

			if err := printLayer(w, value.Layers[i]); err != nil {
				return err
			}
		}

		return nil
	case []string:
		return printLayer(w, value)
	default:
		_, err := fmt.Fprintln(w, value)

		return err
	}
}

// Narrows the keymap to whatever the arguments asked for: the whole thing, one
// layer, or one keycode. Selection is kept apart from encoding so text and JSON
// cannot drift into answering different questions.
func selectKeymap(keymap keymapJSON, args []string) (any, error) {
	if len(args) > 2 {
		return nil, fmt.Errorf("%w: get takes at most a layer and an index", errTooManyArgs)
	}

	if len(args) == 0 {
		return keymap, nil
	}

	layer, err := parseLayer(keymap, args[0])
	if err != nil {
		return nil, err
	}

	if len(args) == 1 {
		return keymap.Layers[layer], nil
	}

	index, err := parseIndex(keymap.Layers[layer], args[1])
	if err != nil {
		return nil, err
	}

	return keymap.Layers[layer][index], nil
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
