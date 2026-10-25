//go:build darwin || windows || linux

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestConformanceOnThisPlatform runs the scenarios in a separate process, since
// a native event loop owns its process's main thread. It opens real windows, so
// it runs only when asked: VIVA_CONFORMANCE=1 go test ./cmd/viva-conformance
func TestConformanceOnThisPlatform(t *testing.T) {
	if os.Getenv("VIVA_CONFORMANCE") == "" {
		t.Skip("set VIVA_CONFORMANCE=1 to open real windows")
	}
	output, err := exec.Command("go", "run", ".").CombinedOutput()
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		t.Log(line)
		if strings.HasPrefix(line, "FAIL") {
			t.Error(line)
		}
	}
	if err != nil {
		t.Fatalf("viva-conformance: %v", err)
	}
}
