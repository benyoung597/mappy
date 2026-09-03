package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	errNoKeyboard      = errors.New("no ZSA keyboard found")
	errManyKeyboards   = errors.New("more than one ZSA keyboard attached")
	errUnknownKeyboard = errors.New("attached ZSA keyboard is not in the table")
)

// Every ZSA board shares this vendor, so the product id alone identifies the
// board and, for the Moonlander, which revision it is.
const zsaVendorID = "3297"

// Taken from usb.pid in the ZSA fork's keyboards/zsa/**/info.json. The vid
// lives in a parent info.json and is inherited, so the per board files look
// like they have no vendor. Regenerate against a firmware tree if a new board
// appears - a missing row reports the product id, so adding one is obvious.
var zsaKeyboards = map[string]string{
	"1969": "zsa/moonlander/reva",
	"1972": "zsa/moonlander/revb",
	"1977": "zsa/voyager",
	"4974": "zsa/ergodox_ez/m32u4/base",
	"4975": "zsa/ergodox_ez/m32u4/shine",
	"4976": "zsa/ergodox_ez/m32u4/glow",
	"2030": "zsa/ergodox_ez/stm32/base",
	"2020": "zsa/ergodox_ez/stm32/shine",
	"2010": "zsa/ergodox_ez/stm32/glow",
	"c6ce": "zsa/planck_ez/base",
	"c6cf": "zsa/planck_ez/glow",
}

// Overridden by tests, which write idVendor and idProduct into a temp tree.
var usbDeviceRoot = "/sys/bus/usb/devices"

// Names the attached keyboard as a qmk target. Reads sysfs rather than shelling
// out to lsusb, which is not guaranteed to be installed.
//
// This reports what the board says it is, which is the truth as long as it was
// flashed with firmware matching its hardware - ZSA's own tooling does that. A
// board already cross flashed with the wrong revision will report the wrong
// answer confidently.
//
// Useless mid flash: the board enters DFU as 0483:df11 and stops answering as
// 3297, so detection has to happen before flashing starts.
func detectKeyboard() (string, error) {
	products, err := attachedZSAProducts()
	if err != nil {
		return "", err
	}

	switch len(products) {
	case 0:
		return "", errNoKeyboard
	case 1:
		keyboard, ok := zsaKeyboards[products[0]]
		if !ok {
			return "", fmt.Errorf("%w: product id %s", errUnknownKeyboard, products[0])
		}

		return keyboard, nil
	default:
		return "", fmt.Errorf("%w: %s", errManyKeyboards, strings.Join(describe(products), ", "))
	}
}

// An explicit -keyboard always wins. Detection is the convenience, not the rule:
// it cannot see a board that is unplugged, and mappy still has to work on a
// keymap.c for hardware that is not to hand.
func resolveKeyboard(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}

	keyboard, err := detectKeyboard()
	if err != nil {
		return "", fmt.Errorf("%w (pass -keyboard to say which)", err)
	}

	return keyboard, nil
}

func attachedZSAProducts() ([]string, error) {
	entries, err := os.ReadDir(usbDeviceRoot)
	if err != nil {
		return nil, err
	}

	var products []string

	for _, entry := range entries {
		device := filepath.Join(usbDeviceRoot, entry.Name())

		vendor, err := readUSBAttr(device, "idVendor")
		if err != nil || vendor != zsaVendorID {
			continue
		}

		product, err := readUSBAttr(device, "idProduct")
		if err != nil {
			continue
		}

		products = append(products, product)
	}

	sort.Strings(products)

	return products, nil
}

// A USB device directory is missing these often enough - hubs, interfaces - that
// an unreadable attribute means "not a match", not "something is wrong".
func readUSBAttr(device, name string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(device, name))
	if err != nil {
		return "", err
	}

	return strings.ToLower(strings.TrimSpace(string(raw))), nil
}

func describe(products []string) []string {
	described := make([]string, 0, len(products))

	for _, product := range products {
		if keyboard, ok := zsaKeyboards[product]; ok {
			described = append(described, fmt.Sprintf("%s (%s)", keyboard, product))

			continue
		}

		described = append(described, product)
	}

	return described
}
