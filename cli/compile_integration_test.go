//go:build compile

// Exercises a real firmware build. Needs qmk, the ZSA fork, the arm toolchain
// and a keymap in the userspace, and a cold build takes minutes, so it is kept
// out of both `make test` and `make integration-test`:
//
//	go test -tags compile ./...
package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCompileCommand(t *testing.T) {
	userspaceKeymap := filepath.Join(
		os.Getenv("HOME"),
		".config/qmk-userspace/keyboards/zsa/moonlander/keymaps/vimster/keymap.c",
	)

	if _, err := os.Stat(userspaceKeymap); err != nil {
		t.Skipf("no userspace keymap to build: %v", err)
	}

	tests := []struct {
		name string
		args []string

		wantErr error
	}{
		{
			name: "builds the keymap named by the file",
			args: []string{"-file", userspaceKeymap},
		},
		{
			name: "builds the keymap named directly",
			args: []string{"-keymap", "vimster"},
		},
		{
			// the point of deriving the name: a scratch file is not buildable
			name:    "a file outside a keymaps directory",
			args:    []string{"-file", "/tmp/keymap.c"},
			wantErr: errKeymapNotInDir,
		},
		{
			name:    "neither a keymap nor a file",
			args:    nil,
			wantErr: errNoKeymapName,
		},
		{
			name:    "a keymap that does not exist",
			args:    []string{"-keymap", "no-such-keymap"},
			wantErr: errCompile,
		},
		{
			name:    "positional arguments are refused",
			args:    []string{"-keymap", "vimster", "0"},
			wantErr: errTooManyArgs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"compile", "-keyboard", fixtureKeyboard}, tt.args...)

			err := run(args, io.Discard)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("run() error = %v, expected %v", err, tt.wantErr)
			}
		})
	}
}
