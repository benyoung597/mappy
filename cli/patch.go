package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

var errPatch = errors.New("cannot patch the firmware tree")

// ZSA's inline asm constrains the operand with "g", which lets the compiler
// pick a memory operand, but msr only accepts a register. GCC 12 happened to
// choose a register and GCC 16 does not, so the tree stops assembling with
//
//	Error: Thumb encoding does not support an immediate here -- `msr msp,[r3]'
//
// The constraint was always wrong. Matching the string rather than a file and
// line matters: ZSA repeated the idiom in more than one board, and line
// numbers move between firmware branches.
const (
	brokenConstraint = `__asm__ volatile("msr msp, %0" ::"g"(`
	fixedConstraint  = `__asm__ volatile("msr msp, %0" ::"r"(`
)

// Only ZSA's boards are searched. The rest of a QMK tree is thousands of
// keyboards mappy has no business rewriting.
const patchRoot = "keyboards/zsa"

func patch(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("patch", flag.ContinueOnError)
	flags.SetOutput(stdout)

	qmkHome := flags.String("qmk-home", "", "firmware tree to patch; the working directory when unset")
	dryRun := flags.Bool("n", false, "list what would change, write nothing")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return err
	}

	if flags.NArg() != 0 {
		return fmt.Errorf("%w: patch takes no arguments", errTooManyArgs)
	}

	tree := *qmkHome

	if tree == "" {
		working, err := os.Getwd()
		if err != nil {
			return err
		}

		tree = working
	}

	broken, err := unpatchedFiles(tree)
	if err != nil {
		return err
	}

	if len(broken) == 0 {
		_, err := fmt.Fprintln(stdout, "nothing to patch")

		return err
	}

	for _, file := range broken {
		if *dryRun {
			if _, err := fmt.Fprintf(stdout, "would patch %s\n", file); err != nil {
				return err
			}

			continue
		}

		if err := applyConstraintFix(file); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(stdout, "patched %s\n", file); err != nil {
			return err
		}
	}

	return nil
}

// Lists the files still carrying the broken constraint. A file already fixed
// does not match, so patching twice changes nothing and an upstream fix makes
// this find nothing at all.
func unpatchedFiles(tree string) ([]string, error) {
	root := filepath.Join(tree, patchRoot)

	if _, err := os.Stat(root); err != nil {
		return nil, fmt.Errorf("%w: %w", errPatch, err)
	}

	var broken []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() || filepath.Ext(path) != ".c" {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if bytes.Contains(src, []byte(brokenConstraint)) {
			broken = append(broken, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errPatch, err)
	}

	return broken, nil
}

func applyConstraintFix(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	fixed := bytes.ReplaceAll(src, []byte(brokenConstraint), []byte(fixedConstraint))

	return os.WriteFile(path, fixed, info.Mode().Perm())
}

// Appended to a failed build. Checking up front would refuse builds that are
// fine - an unpatched ErgoDox file has nothing to do with building a
// Moonlander - so the tree is only inspected once something has already gone
// wrong, where the answer is worth having.
func withPatchHint(qmkHome string, err error) error {
	if err == nil {
		return nil
	}

	tree := qmkHome

	if tree == "" {
		working, wderr := os.Getwd()
		if wderr != nil {
			return err
		}

		tree = working
	}

	broken, scanErr := unpatchedFiles(tree)
	if scanErr != nil || len(broken) == 0 {
		return err
	}

	return fmt.Errorf("%w\n  the firmware tree has %d file(s) a modern GCC cannot assemble; run `mappy patch` to fix them", err, len(broken))
}
