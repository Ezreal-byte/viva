package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Op is one step or expectation of a task, run in order by the check test the tool
// puts into the generated project (check_test.go.txt documents each Op).
type Op struct{ Op, Arg string }

// Task is one tasks/*.md file: a front matter of `key: value` lines between two `---`
// lines, then the request exactly as a user would write it.
type Task struct {
	Name    string // file name without .md
	Title   string
	Needs   string // a capability the framework lacks; such a task is reported, not scored
	Request string
	Ops     []Op     // run inside the project by the check test
	Source  []string // substrings main.go must contain
}

var testOps = map[string]bool{
	"text": true, "not-text": true, "hits": true, "click": true, "click-at": true, "type": true,
	"key": true, "wait": true, "drag": true, "moved": true, "resizable": true, "animates": true,
	"bg-mark": true, "before": true, "right-of": true, "bg-changed": true, "layout": true,
}

func parseTask(name, content string) (Task, error) {
	t := Task{Name: name}
	rest, ok := strings.CutPrefix(content, "---\n")
	if !ok {
		return t, fmt.Errorf("%s: missing front matter", name)
	}
	front, body, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return t, fmt.Errorf("%s: front matter is not closed", name)
	}
	for _, line := range strings.Split(front, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, _ := strings.Cut(line, ":")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		switch {
		case key == "title":
			t.Title = value
		case key == "needs":
			t.Needs = value
		case key == "source":
			t.Source = append(t.Source, value)
		case testOps[key]:
			t.Ops = append(t.Ops, Op{key, value})
		default:
			return t, fmt.Errorf("%s: unknown directive %q", name, key)
		}
	}
	t.Request = strings.TrimSpace(body)
	if t.Title == "" || t.Request == "" {
		return t, fmt.Errorf("%s: needs a title and a request", name)
	}
	return t, nil
}

func loadTasks(dir string) ([]Task, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var tasks []Task
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		t, err := parseTask(strings.TrimSuffix(filepath.Base(p), ".md"), string(data))
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
