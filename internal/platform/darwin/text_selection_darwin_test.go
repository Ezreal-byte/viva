package darwin

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNativeTextSelection(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "text-selection")
	command := exec.Command("xcrun", "clang", "-fno-objc-arc", "-Werror",
		"-Wno-deprecated-declarations", "testdata/text_selection.m", "-o", executable,
		"-framework", "Cocoa", "-framework", "CoreText", "-framework", "Metal", "-framework", "QuartzCore")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile native selection tests: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("native selection tests: %v\n%s", err, output)
	} else {
		t.Logf("%s", output)
	}
}
