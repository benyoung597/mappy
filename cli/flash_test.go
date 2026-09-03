package main

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckFlashTarget(t *testing.T) {
	const (
		reva = "zsa/moonlander/reva"
		revb = "zsa/moonlander/revb"
	)

	tests := []struct {
		name string

		detected  string
		requested string
		force     bool

		wantErr error
	}{
		{
			name:      "nothing requested, so what is attached is what is flashed",
			detected:  reva,
			requested: "",
		},
		{
			name:      "the request agrees with the hardware",
			detected:  reva,
			requested: reva,
		},
		{
			// the hazard: revb firmware boots on reva hardware and types, while
			// writing settings to emulated flash the board does not have
			name:      "the wrong revision is refused",
			detected:  reva,
			requested: revb,
			wantErr:   errWrongBoard,
		},
		{
			name:      "a different board entirely is refused",
			detected:  reva,
			requested: "zsa/voyager",
			wantErr:   errWrongBoard,
		},
		{
			name:      "force allows a deliberate cross flash",
			detected:  reva,
			requested: revb,
			force:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkFlashTarget(tt.detected, tt.requested, tt.force)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("checkFlashTarget() error = %v, expected %v", err, tt.wantErr)
			}
		})
	}
}

// Refusing is only useful if the message says what is attached, what was asked
// for, and how to override it.
func TestCheckFlashTargetNamesBothBoards(t *testing.T) {
	err := checkFlashTarget("zsa/moonlander/reva", "zsa/moonlander/revb", false)

	want := "the attached keyboard is not the one being flashed: " +
		"zsa/moonlander/reva is attached, zsa/moonlander/revb was asked for (pass -force to flash anyway)"

	if err == nil || err.Error() != want {
		t.Fatalf("checkFlashTarget() error = %v, expected %q", err, want)
	}
}

func TestFlashTarget(t *testing.T) {
	tests := []struct {
		name string

		devices   []string
		requested string
		force     bool

		want    string
		wantErr error
	}{
		{
			name:    "detects when nothing is requested",
			devices: []string{"5-1.1:3297:1969"},
			want:    "zsa/moonlander/reva",
		},
		{
			name:      "agreeing request is used",
			devices:   []string{"5-1.1:3297:1969"},
			requested: "zsa/moonlander/reva",
			want:      "zsa/moonlander/reva",
		},
		{
			// unlike get, set and compile, a flag alone is not enough: there
			// has to be a board attached to receive the firmware
			name:      "nothing attached is an error even with a keyboard named",
			requested: "zsa/moonlander/reva",
			wantErr:   errNoBoardToFlash,
		},
		{
			name:      "force flashes a named board with nothing detected",
			requested: "zsa/moonlander/reva",
			force:     true,
			want:      "zsa/moonlander/reva",
		},
		{
			name:    "force with no keyboard named still has nothing to flash",
			force:   true,
			wantErr: errNoBoardToFlash,
		},
		{
			name:      "the wrong revision is refused",
			devices:   []string{"5-1.1:3297:1969"},
			requested: "zsa/moonlander/revb",
			wantErr:   errWrongBoard,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeUSB(t, tt.devices...)

			got, err := flashTarget(tt.requested, tt.force)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("flashTarget() error = %v, expected %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("flashTarget() = %q, expected %q", got, tt.want)
			}
		})
	}
}

// The board stops being a keyboard when it enters the bootloader, so the
// instruction has to be readable before that happens.
func TestResetPromptSaysWhatToPress(t *testing.T) {
	for _, want := range []string{"reset button", "Waiting for bootloader", "pinhole", "left half"} {
		if !strings.Contains(resetPrompt, want) {
			t.Fatalf("the reset prompt does not mention %q", want)
		}
	}
}
