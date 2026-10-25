//go:build !linux

package app

func nativeSetPrimarySelection(string) error  { return ErrUnsupported }
func nativePrimarySelection() (string, error) { return "", ErrUnsupported }
