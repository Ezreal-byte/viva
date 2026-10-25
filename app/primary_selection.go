package app

import (
	"fmt"
	"strings"
)

// SetPrimarySelection makes text the primary selection, the second clipboard of X11
// desktops that selecting text fills without a copy command and a middle click pastes. An
// editor calls it as its selection changes; empty text gives the selection up. Only Linux
// has one: elsewhere it returns ErrUnsupported, so a caller ignores that error.
func SetPrimarySelection(text string) error {
	if strings.IndexByte(text, 0) >= 0 {
		return fmt.Errorf("app: selection text contains NUL")
	}
	if err, handled := driverSetPrimarySelection(text); handled {
		return err
	}
	return nativeSetPrimarySelection(text)
}

// PrimarySelection is the primary selection's text, empty when nothing is selected
// anywhere. An editor reads it on a middle click. It returns ErrUnsupported where there is no
// primary selection.
func PrimarySelection() (string, error) {
	if text, err, handled := driverPrimarySelection(); handled {
		return text, err
	}
	return nativePrimarySelection()
}
