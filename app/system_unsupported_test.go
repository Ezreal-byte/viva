//go:build !windows && !darwin && !linux && !(js && wasm)

package app

import (
	"errors"
	"testing"
)

func TestUnsupportedSystemIntegration(t *testing.T) {
	if err := OpenURL("https://example.com"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("OpenURL error = %v, want ErrUnsupported", err)
	}
}
