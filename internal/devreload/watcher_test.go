package devreload

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherDetectsSourceChangesAndIgnoresBuildOutput(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "main.go")
	if err := os.WriteFile(source, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	watch := newWatcher(root, time.Millisecond)
	before, err := watch.scan()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := watch.scan()
	if err != nil {
		t.Fatal(err)
	}
	if !stampsChanged(before, after) {
		t.Fatal("source edit was not detected")
	}
	if err := os.Mkdir(filepath.Join(root, "build"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "build", "worker.go"), []byte("ignored"), 0o644); err != nil {
		t.Fatal(err)
	}
	ignored, err := watch.scan()
	if err != nil {
		t.Fatal(err)
	}
	if stampsChanged(after, ignored) {
		t.Fatal("build output triggered a source change")
	}
}

func TestModuleRootWalksParents(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "cmd", "app")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := moduleRoot(nested)
	if err != nil || got != root {
		t.Fatalf("module root = %q, err = %v", got, err)
	}
}
