package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --program packages the executable it is given and never runs the go tool: the
// fixture's module could not even build the file named here.
func TestPackageProgramUsesTheGivenExecutableWithoutBuilding(t *testing.T) {
	dir := fixture(t, "[app]\nname = \"Notes\"\nid = \"dev.example.notes\"\nversion = \"1.2.3\"\n")
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("this is not Go"), 0o644); err != nil {
		t.Fatal(err)
	}
	given := filepath.Join(t.TempDir(), "notes-signed.exe")
	content := bytes.Repeat([]byte("MZ-prebuilt-"), 100)
	if err := os.WriteFile(given, content, 0o755); err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	args := []string{"package", "-C", dir, "--target", "windows", "--arch", "amd64", "--format", "zip", "--program", given}
	if err := runCLI(args, &output, io.Discard); err != nil {
		t.Fatalf("%v\n%s", err, output.String())
	}
	reader, err := zip.OpenReader(filepath.Join(dir, "build", "notes-1.2.3-windows-amd64.zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != 1 {
		t.Fatalf("zip holds %d files", len(reader.File))
	}
	file, _ := reader.File[0].Open()
	defer file.Close()
	got, _ := io.ReadAll(file)
	if !bytes.Equal(got, content) {
		t.Fatal("the zip does not hold the program that was given")
	}
	if _, err := os.Stat(given); err != nil {
		t.Fatalf("the given program was removed: %v", err)
	}
}

func TestPackageProgramMustBeAFile(t *testing.T) {
	dir := fixture(t, "[app]\nname = \"Notes\"\nid = \"dev.example.notes\"\n")
	for _, program := range []string{filepath.Join(dir, "missing.exe"), dir} {
		err := runCLI([]string{"package", "-C", dir, "--target", "windows", "--format", "zip", "--program", program}, io.Discard, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "is not a file") {
			t.Errorf("--program %s: error = %v", program, err)
		}
	}
}
