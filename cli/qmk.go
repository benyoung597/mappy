package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	errParsingKeyMap = errors.New("issue parsing keymap json")
	errNoLayout      = errors.New("no layout found in keymap json")
	errRaggedLayers  = errors.New("keymap contains ragged layers")
	errC2JSON        = errors.New("qmk c2json failed")
)

type keymapJSON struct {
	Keyboard string     `json:"keyboard"`
	Keymap   string     `json:"keymap"`
	Layout   string     `json:"layout"`
	Layers   [][]string `json:"layers"`
}

func parseKeymapJSON(raw []byte) (keymapJSON, error) {
	var keymap keymapJSON

	err := json.Unmarshal(raw, &keymap)
	if err != nil {
		return keymapJSON{}, errors.Join(errParsingKeyMap, err)
	}

	if keymap.Layout == "" {
		return keymapJSON{}, errNoLayout
	}

	if len(keymap.Layers) == 0 {
		return keymapJSON{}, errNoLayers
	}

	if len(keymap.Layers[0]) == 0 {
		return keymapJSON{}, errNoLayers
	}

	for i, layer := range keymap.Layers {
		if len(layer) != len(keymap.Layers[0]) {
			return keymapJSON{}, fmt.Errorf("%w: layer %d has %d keycodes, layer 0 has %d",
				errRaggedLayers, i, len(layer), len(keymap.Layers[0]))
		}
	}

	return keymap, nil
}

// Reads a keymap.c through qmk c2json. qmkHome selects the firmware tree by
// becoming the subprocess working directory, which outranks user.qmk_home and
// QMK_HOME in QMK's resolution order - see cli/README.md. Empty means inherit
// ours, which is only safe when the caller knows where it is standing.
//
// --no-cpp leaves #define'd keycodes as symbols rather than expanding them, so
// a layout's own DUAL_FUNC_0 survives the round trip back into keymap.c.
//
// The keyboard is validated by c2json against the tree; name is not, it is
// copied into the "keymap" field of the output and nothing more.
func readKeymap(qmkHome, keyboard, name, path string) (keymapJSON, error) {
	// cmd.Dir moves the subprocess into the firmware tree, so a relative path
	// would resolve against that tree instead of the caller's directory.
	abs, err := filepath.Abs(path)
	if err != nil {
		return keymapJSON{}, errors.Join(errC2JSON, err)
	}

	cmd := exec.Command("qmk", "c2json", "-kb", keyboard, "-km", name, "--no-cpp", abs)
	cmd.Dir = qmkHome

	// Output parks stderr on the ExitError, which is where the reason lives: a
	// failing c2json prints usage to stdout, so stdout is never worth reading.
	stdout, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return keymapJSON{}, fmt.Errorf("%w: %s", errC2JSON, strings.TrimSpace(string(exit.Stderr)))
		}

		return keymapJSON{}, errors.Join(errC2JSON, err)
	}

	return parseKeymapJSON(stdout)
}
