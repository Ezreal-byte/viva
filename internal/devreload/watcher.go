package devreload

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type fileStamp struct {
	modified int64
	size     int64
	mode     fs.FileMode
}

type watcher struct {
	root     string
	interval time.Duration
	changes  chan struct{}
}

func newWatcher(root string, interval time.Duration) *watcher {
	return &watcher{root: root, interval: interval, changes: make(chan struct{}, 1)}
}

func (watcher *watcher) run(ctx context.Context) error {
	known, err := watcher.scan()
	if err != nil {
		return err
	}
	ticker := time.NewTicker(watcher.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			next, scanErr := watcher.scan()
			if scanErr != nil {
				continue
			}
			if stampsChanged(known, next) {
				known = next
				select {
				case watcher.changes <- struct{}{}:
				default:
				}
			}
		}
	}
}

func (watcher *watcher) scan() (map[string]fileStamp, error) {
	result := make(map[string]fileStamp)
	err := filepath.WalkDir(watcher.root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if path != watcher.root && ignoredDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !watchedFile(entry.Name()) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		result[path] = fileStamp{modified: info.ModTime().UnixNano(), size: info.Size(), mode: info.Mode()}
		return nil
	})
	return result, err
}

func stampsChanged(left, right map[string]fileStamp) bool {
	if len(left) != len(right) {
		return true
	}
	for path, stamp := range left {
		if right[path] != stamp {
			return true
		}
	}
	return false
}

func ignoredDirectory(name string) bool {
	switch name {
	case ".git", ".hg", ".svn", ".idea", ".vscode", "build", "node_modules", "vendor":
		return true
	default:
		return strings.HasPrefix(name, ".viva-")
	}
}

func watchedFile(name string) bool {
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, "~") || strings.HasSuffix(name, ".tmp") {
		return false
	}
	extension := strings.ToLower(filepath.Ext(name))
	switch extension {
	case ".go", ".c", ".h", ".m", ".mm", ".s", ".syso", ".metal", ".json", ".toml", ".yaml", ".yml", ".txt", ".png", ".jpg", ".jpeg", ".webp", ".svg":
		return true
	default:
		return name == "go.mod" || name == "go.sum" || name == "go.work" || name == "go.work.sum"
	}
}

func moduleRoot(directory string) (string, error) {
	root, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	for {
		if info, statErr := os.Stat(filepath.Join(root, "go.mod")); statErr == nil && !info.IsDir() {
			return root, nil
		}
		parent := filepath.Dir(root)
		if parent == root {
			return "", os.ErrNotExist
		}
		root = parent
	}
}
