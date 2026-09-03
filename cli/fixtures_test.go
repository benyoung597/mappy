//go:build integration || compile

// Fixtures shared by every tagged test file. Both tags need them, so this file
// carries both: -tags integration and -tags compile each pull it in, and asking
// for both does not declare them twice.
package main

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	fixtureKeyboard = "zsa/moonlander/reva"
	fixtureName     = "vimster"
)

// Copies the fixture somewhere writable so a test can edit it.
func tempKeymap(t *testing.T) string {
	t.Helper()

	src, err := os.ReadFile(fixtureKeymap)
	if err != nil {
		t.Fatalf("failed to read the fixture: %v", err)
	}

	path := filepath.Join(t.TempDir(), "keymap.c")

	if err := os.WriteFile(path, src, 0o644); err != nil {
		t.Fatalf("failed to write the temp keymap: %v", err)
	}

	return path
}
