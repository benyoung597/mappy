//go:build flash

// Exercises qmk flash against a real keyboard, so it needs one attached.
//
// Nothing here writes firmware. A real flash needs someone to press the reset
// button while the command waits, which no automated test can do, so this
// covers the argument assembly through qmk's own dry run and the refusals that
// happen before qmk is reached:
//
//	go test -tags flash ./...
package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestFlashCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string

		wantErr error
	}{
		{
			name: "a dry run against whatever is attached",
			args: []string{"-keymap", "vimster", "-n"},
		},
		{
			name:    "the wrong revision is refused before qmk is reached",
			args:    []string{"-keymap", "vimster", "-keyboard", "zsa/moonlander/revb", "-n"},
			wantErr: errWrongBoard,
		},
		{
			name:    "no keymap to flash",
			args:    []string{"-n"},
			wantErr: errNoKeymapName,
		},
		{
			name:    "positional arguments are refused",
			args:    []string{"-keymap", "vimster", "-n", "0"},
			wantErr: errTooManyArgs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer

			err := run(append([]string{"flash"}, tt.args...), &stdout)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("run() error = %v, expected %v", err, tt.wantErr)
			}

			// a dry run must not have told anyone to press anything
			if strings.Contains(stdout.String(), "reset button") {
				t.Fatalf("a dry run printed the reset prompt")
			}
		})
	}
}
