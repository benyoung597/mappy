package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
)

var (
	errFlash          = errors.New("qmk flash failed")
	errWrongBoard     = errors.New("the attached keyboard is not the one being flashed")
	errNoBoardToFlash = errors.New("no keyboard to flash")
)

const resetPrompt = `Press the reset button when qmk prints "Waiting for bootloader...".
On a Moonlander it is the pinhole on the left half, beside the USB-C port.
The board stops being a keyboard the moment it enters the bootloader, so the
command has to be running before you press it. Unplugging and replugging
returns it to its current firmware if anything goes wrong.`

func flashCmd(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("flash", flag.ContinueOnError)
	flags.SetOutput(stdout)

	qmkHome := flags.String("qmk-home", "", "firmware tree to run qmk in")
	keyboard := flags.String("keyboard", "", "keyboard to flash; detected from USB when unset")
	name := flags.String("keymap", "", "keymap to flash; taken from -file when unset")
	path := flags.String("file", "", "keymap.c whose keymap name should be flashed")
	clean := flags.Bool("clean", false, "remove object files before compiling")
	parallel := flags.Int("j", 0, "parallel make jobs, 0 for unlimited")
	force := flags.Bool("force", false, "flash even if the attached keyboard is a different target")
	dryRun := flags.Bool("n", false, "show the command qmk would run, build and flash nothing")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return err
	}

	if flags.NArg() != 0 {
		return fmt.Errorf("%w: flash takes no arguments", errTooManyArgs)
	}

	keymap, err := resolveKeymapName(*name, *path)
	if err != nil {
		return err
	}

	board, err := flashTarget(*keyboard, *force)
	if err != nil {
		return err
	}

	if !*dryRun {
		if _, err := fmt.Fprintln(stdout, resetPrompt); err != nil {
			return err
		}
	}

	return runFlash(*qmkHome, board, keymap, *clean, *parallel, *dryRun, stdout)
}

// Unlike the other commands, a flash cannot fall back to the flag alone: there
// has to be a board attached to receive it, and it has to be the right one.
//
// reva and revb are separate targets with different bootloaders and different
// flash addresses. Rev B firmware on rev A hardware boots and types while
// writing its settings to emulated flash the hardware does not have - not
// damaging, but wrong, and the board reports the truth about itself right up
// until it is cross flashed.
func flashTarget(requested string, force bool) (string, error) {
	detected, err := detectKeyboard()
	if err != nil {
		if force && requested != "" {
			return requested, nil
		}

		return "", fmt.Errorf("%w: %w", errNoBoardToFlash, err)
	}

	if err := checkFlashTarget(detected, requested, force); err != nil {
		return "", err
	}

	if requested != "" {
		return requested, nil
	}

	return detected, nil
}

func checkFlashTarget(detected, requested string, force bool) error {
	if requested == "" || requested == detected || force {
		return nil
	}

	return fmt.Errorf("%w: %s is attached, %s was asked for (pass -force to flash anyway)",
		errWrongBoard, detected, requested)
}

// Streams qmk's output. dfu-util reports
//
//	dfuERROR, status(10) = Device's firmware is corrupt
//
// before it writes anything - that is benign, it clears the status and carries
// on, so nothing here should start matching on the output and calling it a
// failure. Only the exit code says whether the flash worked.
func runFlash(qmkHome, keyboard, keymap string, clean bool, parallel int, dryRun bool, stdout io.Writer) error {
	args := []string{"flash", "-kb", keyboard, "-km", keymap}

	if dryRun {
		args = append(args, "-n")
	}

	if clean {
		args = append(args, "--clean")
	}

	if parallel != 0 {
		args = append(args, "-j", strconv.Itoa(parallel))
	}

	cmd := exec.Command("qmk", args...)
	cmd.Dir = qmkHome
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return withPatchHint(qmkHome, fmt.Errorf("%w: %s %s: %w", errFlash, keyboard, keymap, err))
	}

	return nil
}
