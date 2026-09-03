package main

import (
	"errors"
	"testing"
)

func TestKeymapNameFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string

		want    string
		wantErr error
	}{
		{
			name: "a keymap in a userspace",
			path: "/home/x/.config/qmk-userspace/keyboards/zsa/moonlander/keymaps/vimster/keymap.c",
			want: "vimster",
		},
		{
			name: "a relative path",
			path: "keyboards/zsa/moonlander/keymaps/vimster/keymap.c",
			want: "vimster",
		},
		{
			name: "the file need not be called keymap.c",
			path: "keymaps/default/anything.c",
			want: "default",
		},
		{
			name: "redundant separators are cleaned first",
			path: "keymaps//vimster//keymap.c",
			want: "vimster",
		},
		{
			// the trap this exists to make loud: an edited scratch file is not
			// a keymap qmk can build
			name:    "a path outside a keymaps directory",
			path:    "/tmp/keymap.c",
			wantErr: errKeymapNotInDir,
		},
		{
			name:    "a bare filename",
			path:    "keymap.c",
			wantErr: errKeymapNotInDir,
		},
		{
			name:    "the parent is not named keymaps",
			path:    "keyboards/zsa/moonlander/layouts/vimster/keymap.c",
			wantErr: errKeymapNotInDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := keymapNameFromPath(tt.path)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("keymapNameFromPath(%q) error = %v, expected %v", tt.path, err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("keymapNameFromPath(%q) = %q, expected %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestResolveKeymapName(t *testing.T) {
	tests := []struct {
		name string

		keymap string
		path   string

		want    string
		wantErr error
	}{
		{
			name:   "an explicit keymap wins",
			keymap: "default",
			path:   "keymaps/vimster/keymap.c",
			want:   "default",
		},
		{
			name:   "an explicit keymap needs no file",
			keymap: "vimster",
			want:   "vimster",
		},
		{
			name: "the file supplies the name",
			path: "keymaps/vimster/keymap.c",
			want: "vimster",
		},
		{
			name:    "neither is given",
			wantErr: errNoKeymapName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveKeymapName(tt.keymap, tt.path)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("resolveKeymapName() error = %v, expected %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("resolveKeymapName() = %q, expected %q", got, tt.want)
			}
		})
	}
}
