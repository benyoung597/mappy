package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Builds a fake sysfs tree. Each device is "name:vendor:product", and an empty
// vendor or product leaves that attribute absent, the way a hub or an interface
// directory does.
func fakeUSB(t *testing.T, devices ...string) {
	t.Helper()

	root := t.TempDir()

	for _, device := range devices {
		parts := strings.Split(device, ":")

		dir := filepath.Join(root, parts[0])
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to build the fake device: %v", err)
		}

		for i, attr := range []string{"idVendor", "idProduct"} {
			if parts[i+1] == "" {
				continue
			}

			if err := os.WriteFile(filepath.Join(dir, attr), []byte(parts[i+1]+"\n"), 0o644); err != nil {
				t.Fatalf("failed to write %s: %v", attr, err)
			}
		}
	}

	original := usbDeviceRoot
	usbDeviceRoot = root

	t.Cleanup(func() { usbDeviceRoot = original })
}

func TestDetectKeyboard(t *testing.T) {
	tests := []struct {
		name    string
		devices []string

		want    string
		wantErr error
	}{
		{
			name:    "a moonlander rev a",
			devices: []string{"5-1.1:3297:1969"},
			want:    "zsa/moonlander/reva",
		},
		{
			// the revisions are separate targets and not interchangeable
			name:    "a moonlander rev b",
			devices: []string{"5-1.1:3297:1972"},
			want:    "zsa/moonlander/revb",
		},
		{
			name:    "a voyager",
			devices: []string{"1-2:3297:1977"},
			want:    "zsa/voyager",
		},
		{
			name:    "sysfs writes lowercase, but uppercase is read the same",
			devices: []string{"1-2:3297:C6CE"},
			want:    "zsa/planck_ez/base",
		},
		{
			name:    "nothing attached",
			devices: nil,
			wantErr: errNoKeyboard,
		},
		{
			name:    "only devices from other vendors",
			devices: []string{"1-1:1d6b:0002", "1-2:046d:c52b"},
			wantErr: errNoKeyboard,
		},
		{
			// hubs and interface directories have no idVendor at all
			name:    "directories without the attributes are skipped",
			devices: []string{"1-1:0:0", "5-1.1:3297:1969"},
			want:    "zsa/moonlander/reva",
		},
		{
			name:    "a zsa board that is not in the table",
			devices: []string{"1-2:3297:dead"},
			wantErr: errUnknownKeyboard,
		},
		{
			name:    "two zsa boards",
			devices: []string{"1-2:3297:1969", "1-3:3297:1977"},
			wantErr: errManyKeyboards,
		},
		{
			name:    "a zsa board next to other vendors",
			devices: []string{"1-1:1d6b:0002", "5-1.1:3297:1969", "1-2:046d:c52b"},
			want:    "zsa/moonlander/reva",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeUSB(t, tt.devices...)

			got, err := detectKeyboard()

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("detectKeyboard() error = %v, expected %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("detectKeyboard() = %q, expected %q", got, tt.want)
			}
		})
	}
}

// The product id has to reach the message, or adding the missing row means
// going and reading lsusb yourself.
func TestDetectKeyboardErrorsNameWhatItSaw(t *testing.T) {
	tests := []struct {
		name    string
		devices []string

		want string
	}{
		{
			name:    "an unknown board names its product id",
			devices: []string{"1-2:3297:dead"},
			want:    "attached ZSA keyboard is not in the table: product id dead",
		},
		{
			name:    "two boards name both",
			devices: []string{"1-2:3297:1969", "1-3:3297:1977"},
			want:    "more than one ZSA keyboard attached: zsa/moonlander/reva (1969), zsa/voyager (1977)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeUSB(t, tt.devices...)

			_, err := detectKeyboard()

			if err == nil || err.Error() != tt.want {
				t.Fatalf("detectKeyboard() error = %v, expected %q", err, tt.want)
			}
		})
	}
}

func TestResolveKeyboard(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		devices []string

		want    string
		wantErr error
	}{
		{
			// mappy still has to edit a keymap.c for hardware that is elsewhere
			name: "an explicit keyboard wins with nothing attached",
			flag: "zsa/moonlander/revb",
			want: "zsa/moonlander/revb",
		},
		{
			name:    "an explicit keyboard wins over what is attached",
			flag:    "zsa/voyager",
			devices: []string{"5-1.1:3297:1969"},
			want:    "zsa/voyager",
		},
		{
			name:    "no flag detects",
			devices: []string{"5-1.1:3297:1969"},
			want:    "zsa/moonlander/reva",
		},
		{
			name:    "no flag and nothing attached says how to proceed",
			wantErr: errNoKeyboard,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeUSB(t, tt.devices...)

			got, err := resolveKeyboard(tt.flag)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("resolveKeyboard() error = %v, expected %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("resolveKeyboard() = %q, expected %q", got, tt.want)
			}

			if tt.wantErr != nil && !strings.Contains(err.Error(), "pass -keyboard") {
				t.Fatalf("error does not say how to proceed: %v", err)
			}
		})
	}
}
