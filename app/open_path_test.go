package app

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenPathRefusesWhatTheSystemWouldRun(t *testing.T) {
	cases := []struct {
		goos, name string
		mode       fs.FileMode
		refused    bool
	}{
		{"windows", `C:\tmp\notes.TXT`, 0o666, false},
		{"windows", `C:\tmp\setup.EXE`, 0o666, true},
		{"windows", `C:\tmp\run.cmd`, 0o666, true},
		{"windows", `C:\tmp\link.lnk`, 0o666, true},
		{"windows", `C:\tmp\script.ps1`, 0o666, true},
		{"windows", `C:\tmp\main.go`, 0o666, false},
		{"darwin", "/tmp/notes.txt", 0o644, false},
		{"darwin", "/tmp/x.command", 0o644, true},
		{"darwin", "/tmp/x.terminal", 0o644, true},
		{"darwin", "/tmp/notes.txt", 0o755, true},
		{"darwin", "/tmp/script.js", 0o644, false},
		{"linux", "/tmp/a.desktop", 0o644, true},
		{"linux", "/tmp/a.AppImage", 0o644, true},
		{"linux", "/tmp/a.txt", 0o700, true},
		{"linux", "/tmp/a.sh", 0o644, false},
	}
	for _, test := range cases {
		err := checkOpenPath(test.goos, test.name, test.mode)
		if test.refused != errors.Is(err, ErrExecutablePath) {
			t.Errorf("%s %s %v: %v", test.goos, test.name, test.mode, err)
		}
	}
	if err := checkOpenPath("linux", "/tmp/dir", fs.ModeDir|0o755); err == nil || errors.Is(err, ErrExecutablePath) {
		t.Errorf("a directory: %v", err)
	}
	if err := checkOpenPath("linux", "/tmp/link", fs.ModeSymlink|0o777); err == nil {
		t.Error("a symlink was accepted")
	}
}

func TestOpenPathTargetIsAFileURLOnlyWhereTheOpenerTakesURLs(t *testing.T) {
	if got := openPathTarget("darwin", "/tmp/my notes #1.txt"); got != "file:///tmp/my%20notes%20%231.txt" {
		t.Errorf("darwin target %q", got)
	}
	if got := openPathTarget("linux", "/tmp/my notes.txt"); got != "/tmp/my notes.txt" {
		t.Errorf("linux target %q", got)
	}
}

func TestOpenPathValidatesBeforeAskingTheSystem(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"relative.txt", filepath.Join(dir, "missing.txt"), dir, "/tmp/a\x00b"} {
		if err := OpenPath(path); err == nil {
			t.Errorf("%q was accepted", path)
		}
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(file, link); err == nil {
		if err := OpenPath(link); err == nil {
			t.Error("a symbolic link was accepted")
		}
	}
}
