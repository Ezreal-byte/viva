package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed preview/*
var assets embed.FS

func main() {
	output := flag.String("out", "build/docs-data", "generated documentation data directory")
	check := flag.Bool("check", false, "validate the catalog and source without generating files")
	flag.Parse()
	if err := run(*output, *check); err != nil {
		fmt.Fprintln(os.Stderr, "viva docs:", err)
		os.Exit(1)
	}
}

func run(output string, check bool) error {
	root, err := findRoot()
	if err != nil {
		return err
	}
	pages, manifest, err := loadPages(root)
	if err != nil {
		return err
	}
	if check {
		api, err := loadAPI(root)
		if err != nil {
			return err
		}
		if err := validateAPI(api); err != nil {
			return err
		}
		fmt.Printf("validated %d documented components and %d API packages · source %s\n", len(pages), len(api.Packages), manifest.Revision)
		return nil
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}
	if err := validateOutput(root, output); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(filepath.Dir(output), ".docs-staging-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	if err := writeData(root, staging, pages, manifest); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "build", "-trimpath", "-ldflags=-s -w -buildid=", "-o", filepath.Join(staging, "public", "preview", "preview-"+manifest.Revision+".wasm"), "./cmd/component-preview")
	command.Dir = root
	command.Env = targetEnvironment(os.Environ())
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("compile shared Web preview: %w", err)
	}
	gorootBytes, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return err
	}
	wasmRuntime, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(gorootBytes)), "lib", "wasm", "wasm_exec.js"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(staging, "public", "preview", "wasm_exec.js"), wasmRuntime, 0o644); err != nil {
		return err
	}
	current, err := sourceManifest(root)
	if err != nil {
		return err
	}
	if current.Revision != manifest.Revision {
		return errors.New("source changed during generation; keeping the previous preview, run again")
	}
	if err := retainPreviousPreview(output, staging, manifest.Revision); err != nil {
		return err
	}
	if err := publishData(staging, output); err != nil {
		return err
	}
	fmt.Printf("generated %d component documents and shared WASM in %s · source %s\n", len(pages), output, manifest.Revision)
	return nil
}

func retainPreviousPreview(output, staging, currentRevision string) error {
	data, err := os.ReadFile(filepath.Join(output, "catalog.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var previous struct {
		Revision string `json:"revision"`
	}
	if json.Unmarshal(data, &previous) != nil || previous.Revision == "" || previous.Revision == currentRevision {
		return nil
	}
	source, err := os.Open(filepath.Join(output, "public", "preview", "preview-"+previous.Revision+".wasm"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := os.Create(filepath.Join(staging, "public", "preview", "preview-"+previous.Revision+".wasm"))
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func targetEnvironment(environment []string) []string {
	result := make([]string, 0, len(environment)+3)
	for _, entry := range environment {
		if !strings.HasPrefix(entry, "GOOS=") && !strings.HasPrefix(entry, "GOARCH=") && !strings.HasPrefix(entry, "CGO_ENABLED=") {
			result = append(result, entry)
		}
	}
	return append(result, "GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0")
}

func findRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		module, err := os.ReadFile(filepath.Join(directory, "go.mod"))
		if err == nil && strings.HasPrefix(string(module), "module github.com/vivaui/viva\n") {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", errors.New("run inside the Viva repository")
		}
		directory = parent
	}
}

func validateOutput(root, output string) error {
	relative, err := filepath.Rel(filepath.Join(root, "build"), output)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("output must be a subdirectory of this repository's build/ directory")
	}
	for directory := output; directory != root; directory = filepath.Dir(directory) {
		info, err := os.Lstat(directory)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return errors.New("output path must not contain symlinks")
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func publishData(staging, output string) error {
	backup := staging + "-previous"
	if _, err := os.Stat(output); err == nil {
		if _, err := os.Stat(filepath.Join(output, "catalog.json")); err != nil {
			return errors.New("refusing to replace a directory not owned by the docs generator")
		}
		if err := os.Rename(output, backup); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(staging, output); err != nil {
		if _, statErr := os.Stat(backup); statErr == nil {
			if restoreErr := os.Rename(backup, output); restoreErr != nil {
				return fmt.Errorf("publish failed: %v; restore from %s: %w", err, backup, restoreErr)
			}
		}
		return err
	}
	return os.RemoveAll(backup)
}
