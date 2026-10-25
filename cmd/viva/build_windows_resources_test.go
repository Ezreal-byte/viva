package main

import (
	"bytes"
	"debug/pe"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

func buildWindowsFixture(t *testing.T, dir string) (program []byte, report string) {
	t.Helper()
	out := filepath.Join(dir, "out", "fixture.exe")
	var stderr bytes.Buffer
	if err := runCLI([]string{"build", "-C", dir, "--target", "windows", "--arch", "amd64", "-o", out}, io.Discard, &stderr); err != nil {
		t.Fatalf("%v\n%s", err, stderr.String())
	}
	program, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return program, stderr.String()
}

func TestWindowsBuildCarriesTheIconAndVersionInsideTheExe(t *testing.T) {
	dir := fixture(t, "[app]\nname = \"Fixture Notes\"\nid = \"dev.example.notes\"\nversion = \"2.0.1\"\nicon = \"icon.png\"\n")
	writeIcon(t, filepath.Join(dir, "icon.png"), 256)
	program, _ := buildWindowsFixture(t, dir)

	file, err := pe.NewFile(bytes.NewReader(program))
	if err != nil {
		t.Fatalf("the program is not a PE file: %v", err)
	}
	section := file.Section(".rsrc")
	if section == nil {
		t.Fatal("the program has no resources: Explorer would show the default icon")
	}
	if directory := file.OptionalHeader.(*pe.OptionalHeader64).DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE]; directory.VirtualAddress != section.VirtualAddress {
		t.Error("the resource directory does not point at the section")
	}
	data, _ := section.Data()
	if !bytes.Contains(data, []byte("\x89PNG")) {
		t.Error("the icon's images are not in the resources")
	}
	if !bytes.Contains(program, []byte("F\x00i\x00x\x00t\x00u\x00r\x00e\x00 \x00N\x00o\x00t\x00e\x00s\x00")) {
		t.Error("the product name is not in the version information")
	}
	// Task Manager and the file's properties show the file's real name, whatever
	// scratch name the packager built it under.
	if !bytes.Contains(program, utf16Bytes("fixture-notes.exe")) {
		t.Error("the original file name, with .exe, is not in the version information")
	}
}

func TestAnUnusableIconCostsTheProgramItsIconNotTheBuild(t *testing.T) {
	dir := fixture(t, "[app]\nname = \"Fixture Notes\"\nid = \"dev.example.notes\"\nicon = \"icon.png\"\n")
	writeIcon(t, filepath.Join(dir, "icon.png"), 64)
	program, report := buildWindowsFixture(t, dir)
	if !strings.Contains(report, "no icon") {
		t.Errorf("the build did not say the icon was skipped:\n%s", report)
	}
	file, err := pe.NewFile(bytes.NewReader(program))
	if err != nil {
		t.Fatal(err)
	}
	if file.Section(".rsrc") == nil {
		t.Error("the version information was lost along with the icon")
	}
}

func utf16Bytes(text string) []byte {
	var out []byte
	for _, unit := range utf16.Encode([]rune(text)) {
		out = append(out, byte(unit), byte(unit>>8))
	}
	return out
}
