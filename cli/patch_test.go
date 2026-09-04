package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	brokenSource = "void mcu_reset(void) {\n" +
		"    " + brokenConstraint + "*(volatile uint32_t *)APP_ADDRESS));\n" +
		"}\n"
	fixedSource = "void mcu_reset(void) {\n" +
		"    " + fixedConstraint + "*(volatile uint32_t *)APP_ADDRESS));\n" +
		"}\n"
)

// Builds a firmware tree. Each entry is "relative/path.c:broken|fixed|other".
func fakeTree(t *testing.T, files ...string) string {
	t.Helper()

	tree := t.TempDir()

	for _, entry := range files {
		path, kind, _ := strings.Cut(entry, ":")

		full := filepath.Join(tree, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("failed to build the fake tree: %v", err)
		}

		body := map[string]string{
			"broken": brokenSource,
			"fixed":  fixedSource,
			"other":  "int main(void) { return 0; }\n",
		}[kind]

		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	return tree
}

func TestUnpatchedFiles(t *testing.T) {
	tests := []struct {
		name  string
		files []string

		want    []string
		wantErr error
	}{
		{
			name:  "a broken file is found",
			files: []string{"keyboards/zsa/moonlander/moonlander.c:broken"},
			want:  []string{"keyboards/zsa/moonlander/moonlander.c"},
		},
		{
			// ZSA repeated the idiom, so this is not a single file fix
			name: "every broken file is found, however deep",
			files: []string{
				"keyboards/zsa/moonlander/moonlander.c:broken",
				"keyboards/zsa/ergodox_ez/stm32/stm32.c:broken",
			},
			want: []string{
				"keyboards/zsa/ergodox_ez/stm32/stm32.c",
				"keyboards/zsa/moonlander/moonlander.c",
			},
		},
		{
			name:  "an already fixed file does not match",
			files: []string{"keyboards/zsa/moonlander/moonlander.c:fixed"},
		},
		{
			name: "only the broken one of a pair is reported",
			files: []string{
				"keyboards/zsa/moonlander/moonlander.c:fixed",
				"keyboards/zsa/ergodox_ez/stm32/stm32.c:broken",
			},
			want: []string{"keyboards/zsa/ergodox_ez/stm32/stm32.c"},
		},
		{
			name:  "unrelated sources are ignored",
			files: []string{"keyboards/zsa/moonlander/matrix.c:other"},
		},
		{
			// the rest of a QMK tree is not mappy's to rewrite
			name:  "boards outside keyboards/zsa are left alone",
			files: []string{"keyboards/planck/rev6/planck.c:broken", "keyboards/zsa/voyager/voyager.c:fixed"},
		},
		{
			name:    "a tree with no zsa boards at all",
			files:   []string{"keyboards/planck/rev6/planck.c:other"},
			wantErr: errPatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := fakeTree(t, tt.files...)

			got, err := unpatchedFiles(tree)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("unpatchedFiles() error = %v, expected %v", err, tt.wantErr)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("unpatchedFiles() = %v, expected %v", got, tt.want)
			}

			for i, want := range tt.want {
				if rel, _ := filepath.Rel(tree, got[i]); rel != want {
					t.Fatalf("unpatchedFiles()[%d] = %s, expected %s", i, rel, want)
				}
			}
		})
	}
}

func TestApplyConstraintFix(t *testing.T) {
	tree := fakeTree(t, "keyboards/zsa/moonlander/moonlander.c:broken")
	path := filepath.Join(tree, "keyboards/zsa/moonlander/moonlander.c")

	if err := applyConstraintFix(path); err != nil {
		t.Fatalf("applyConstraintFix() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read back: %v", err)
	}

	if string(got) != fixedSource {
		t.Fatalf("applyConstraintFix() produced %q, expected %q", string(got), fixedSource)
	}

	// running it again must change nothing
	if err := applyConstraintFix(path); err != nil {
		t.Fatalf("second applyConstraintFix() error = %v", err)
	}

	again, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read back: %v", err)
	}

	if string(again) != fixedSource {
		t.Fatalf("patching twice changed the file")
	}

	broken, err := unpatchedFiles(tree)
	if err != nil {
		t.Fatalf("unpatchedFiles() error = %v", err)
	}

	if len(broken) != 0 {
		t.Fatalf("the file still reports as unpatched: %v", broken)
	}
}

func TestApplyConstraintFixKeepsTheMode(t *testing.T) {
	tree := fakeTree(t, "keyboards/zsa/moonlander/moonlander.c:broken")
	path := filepath.Join(tree, "keyboards/zsa/moonlander/moonlander.c")

	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}

	if err := applyConstraintFix(path); err != nil {
		t.Fatalf("applyConstraintFix() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, expected 0640", info.Mode().Perm())
	}
}

func TestPatchCommand(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		args  []string

		want    string
		changed bool
	}{
		{
			name:    "patches and says so",
			files:   []string{"keyboards/zsa/moonlander/moonlander.c:broken"},
			want:    "patched",
			changed: true,
		},
		{
			name:  "a dry run writes nothing",
			files: []string{"keyboards/zsa/moonlander/moonlander.c:broken"},
			args:  []string{"-n"},
			want:  "would patch",
		},
		{
			name:  "a clean tree",
			files: []string{"keyboards/zsa/moonlander/moonlander.c:fixed"},
			want:  "nothing to patch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := fakeTree(t, tt.files...)
			path := filepath.Join(tree, "keyboards/zsa/moonlander/moonlander.c")

			var stdout strings.Builder

			args := append([]string{"-qmk-home", tree}, tt.args...)

			if err := patch(args, &stdout); err != nil {
				t.Fatalf("patch() error = %v", err)
			}

			if !strings.Contains(stdout.String(), tt.want) {
				t.Fatalf("patch() wrote %q, expected it to mention %q", stdout.String(), tt.want)
			}

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read back: %v", err)
			}

			if tt.changed && string(after) != fixedSource {
				t.Fatalf("the file was not patched")
			}

			if !tt.changed && strings.Contains(stdout.String(), "would patch") && string(after) != brokenSource {
				t.Fatalf("a dry run wrote to the file")
			}
		})
	}
}

func TestWithPatchHint(t *testing.T) {
	failure := errors.New("qmk compile failed")

	tests := []struct {
		name  string
		files []string
		err   error

		wantHint bool
	}{
		{
			name:     "a broken tree earns the hint",
			files:    []string{"keyboards/zsa/moonlander/moonlander.c:broken"},
			err:      failure,
			wantHint: true,
		},
		{
			// the build failed for some other reason, so do not misdirect
			name:  "a clean tree does not",
			files: []string{"keyboards/zsa/moonlander/moonlander.c:fixed"},
			err:   failure,
		},
		{
			name:  "a success is left alone",
			files: []string{"keyboards/zsa/moonlander/moonlander.c:broken"},
			err:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := fakeTree(t, tt.files...)

			got := withPatchHint(tree, tt.err)

			if tt.err == nil {
				if got != nil {
					t.Fatalf("withPatchHint() = %v, expected nil", got)
				}

				return
			}

			if !errors.Is(got, failure) {
				t.Fatalf("withPatchHint() lost the underlying error: %v", got)
			}

			if strings.Contains(got.Error(), "mappy patch") != tt.wantHint {
				t.Fatalf("withPatchHint() = %q, wantHint = %v", got.Error(), tt.wantHint)
			}
		})
	}
}
