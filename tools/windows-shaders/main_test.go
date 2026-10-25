package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateManifest(t *testing.T) {
	output := t.TempDir()
	source := []byte("shader source")
	var calls int
	compile := func(data []byte, name, profile string) ([]byte, error) {
		if !bytes.Equal(data, source) || name != entries[calls].Name || profile != entries[calls].Profile {
			t.Fatal("unexpected compilation input")
		}
		calls++
		return append([]byte("DXBC"), bytes.Repeat([]byte{byte(calls)}, 28)...), nil
	}
	if err := generate(source, output, compile); err != nil {
		t.Fatal(err)
	}
	encoded, err := os.ReadFile(filepath.Join(output, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest shaderManifest
	if err := json.Unmarshal(encoded, &manifest); err != nil {
		t.Fatal(err)
	}
	if calls != len(entries) || manifest.SourceSHA256 != digest(source) || manifest.Flags != compilerFlags || len(manifest.Entries) != len(entries) {
		t.Fatalf("invalid manifest: %+v", manifest)
	}
	for _, entry := range manifest.Entries {
		code, err := os.ReadFile(filepath.Join(output, entry.Name+".dxbc"))
		if err != nil {
			t.Fatal(err)
		}
		if entry.Bytes != len(code) || entry.SHA256 != digest(code) {
			t.Fatalf("unverified bytecode: %+v", entry)
		}
	}
	if err := generate(source, output, compileShaderFailure); err == nil {
		t.Fatal("compilation failure accepted")
	}
	unchanged, err := os.ReadFile(filepath.Join(output, "manifest.json"))
	if err != nil || !bytes.Equal(unchanged, encoded) {
		t.Fatal("failed compilation changed the manifest")
	}
}

func compileShaderFailure([]byte, string, string) ([]byte, error) {
	return nil, errors.New("invalid shader")
}

func TestGenerateRejectsInvalidBytecode(t *testing.T) {
	output := filepath.Join(t.TempDir(), "absent")
	if err := generate([]byte("source"), output, func([]byte, string, string) ([]byte, error) { return []byte("not DXBC"), nil }); err == nil {
		t.Fatal("invalid bytecode accepted")
	}
	if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("invalid compilation created output")
	}
}
