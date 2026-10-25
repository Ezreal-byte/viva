// Command aieval measures whether a coding model can write a working Viva page from the
// guide a generated project ships (cmd/viva/initfiles/AGENTS.md.txt) and a plain request.
//
// For every task in tasks/*.md, model and run it makes a project with `viva init`, asks the
// model for main.go, runs `go vet` and feeds compiler errors back for a few repair rounds,
// then drives the compiled page headless and checks the task's expectations. See README.md.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type listFlag []string

func (l *listFlag) String() string     { return strings.Join(*l, " ") }
func (l *listFlag) Set(v string) error { *l = append(*l, v); return nil }

func main() {
	var modelFlags listFlag
	flag.Var(&modelFlags, "model", "model as name=baseurl,keyenv[,extra-json]; repeatable")
	configPath := flag.String("config", "~/.config/viva-aieval.toml", "model config file (used when it exists)")
	only := flag.String("only", "", "comma-separated model names to use (default: all configured)")
	taskFilter := flag.String("task", "", "comma-separated task names (default: all)")
	tasksDir := flag.String("tasks", "", "task directory (default: tools/aieval/tasks of the Viva checkout)")
	runs := flag.Int("runs", 1, "samples per task and model")
	repairs := flag.Int("repairs", 3, "repair rounds after a compile failure")
	jobs := flag.Int("j", 4, "parallel runs")
	out := flag.String("out", "", "results directory (default: a new directory under the system temp dir)")
	vivaFlag := flag.String("viva", "", "Viva checkout (default: the nearest directory above the working directory that is module github.com/vivaui/viva)")
	png := flag.Bool("png", false, "also save a screenshot of each compiled page (builds with -tags vivahead)")
	list := flag.Bool("list", false, "list the tasks and exit")
	reportFrom := flag.String("report", "", "print the table for an earlier results.json and exit")
	flag.Parse()
	if *reportFrom != "" {
		if err := reportFile(*reportFrom); err != nil {
			fatal(err)
		}
		return
	}

	root, err := vivaSource(*vivaFlag)
	if err != nil {
		fatal(err)
	}
	dir := *tasksDir
	if dir == "" {
		dir = filepath.Join(root, "tools", "aieval", "tasks")
	}
	tasks, err := loadTasks(dir)
	if err != nil {
		fatal(err)
	}
	tasks = filterTasks(tasks, *taskFilter)
	if *list {
		for _, t := range tasks {
			fmt.Printf("%-14s %s%s\n", t.Name, t.Title, map[bool]string{true: "   [needs: " + t.Needs + "]"}[t.Needs != ""])
		}
		return
	}

	models, err := collectModels(*configPath, modelFlags, *only)
	if err != nil {
		fatal(err)
	}
	keys := make([]string, len(models))
	var secrets []string
	for i, m := range models {
		if keys[i], err = m.key(); err != nil {
			fatal(err)
		}
		secrets = append(secrets, keys[i])
	}
	scrub := func(s string) string {
		for _, k := range secrets {
			if len(k) >= 6 {
				s = strings.ReplaceAll(s, k, "[key]")
			}
		}
		return s
	}

	if *out == "" {
		*out = filepath.Join(os.TempDir(), "viva-aieval", time.Now().Format("20060102-150405"))
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatal(err)
	}
	bin := filepath.Join(*out, "viva")
	if o, err := sh(root, 5*time.Minute, nil, "go", "build", "-o", bin, "./cmd/viva"); err != nil {
		fatal(fmt.Errorf("building viva: %v\n%s", err, o))
	}
	e := &env{vivaRoot: root, vivaBin: bin, out: *out, repairs: *repairs, png: *png, scrub: scrub}

	type job struct {
		task  Task
		model int
		index int
	}
	var queue []job
	for _, t := range tasks {
		for mi := range models {
			for i := 1; i <= *runs; i++ {
				queue = append(queue, job{t, mi, i})
			}
		}
	}
	results := make([]Run, len(queue))
	var wg sync.WaitGroup
	slots := make(chan struct{}, max(*jobs, 1))
	for i, j := range queue {
		wg.Add(1)
		slots <- struct{}{}
		go func() {
			defer func() { <-slots; wg.Done() }()
			r := e.runJob(j.task, models[j.model], keys[j.model], j.index)
			results[i] = r
			fmt.Fprintf(os.Stderr, "%-14s %-18s run %d: %s\n", r.Task, r.Model, r.Index, verdict(r))
		}()
	}
	wg.Wait()

	summary := summarize(tasks, models, results)
	data, _ := json.MarshalIndent(struct {
		Runs    []Run
		Summary []Stat
	}{results, summary}, "", " ")
	if err := os.WriteFile(filepath.Join(*out, "results.json"), []byte(scrub(string(data))), 0o644); err != nil {
		fatal(err)
	}
	table := renderTable(tasks, models, summary, results)
	os.WriteFile(filepath.Join(*out, "table.md"), []byte(table), 0o644)
	fmt.Print(table)
	fmt.Fprintf(os.Stderr, "\nresults in %s (raw replies and projects under raw/)\n", *out)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "aieval:", err)
	os.Exit(1)
}

func filterTasks(tasks []Task, filter string) []Task {
	if filter == "" {
		return tasks
	}
	want := map[string]bool{}
	for _, n := range strings.Split(filter, ",") {
		want[strings.TrimSpace(n)] = true
	}
	var out []Task
	for _, t := range tasks {
		if want[t.Name] {
			out = append(out, t)
		}
	}
	return out
}

func collectModels(configPath string, flags listFlag, only string) ([]Model, error) {
	var models []Model
	if _, err := os.Stat(expandHome(configPath)); err == nil {
		ms, err := loadConfig(expandHome(configPath))
		if err != nil {
			return nil, err
		}
		models = append(models, ms...)
	}
	if m, ok := envModel(); ok {
		models = append(models, m)
	}
	for _, f := range flags {
		m, err := parseModelFlag(f)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	if only != "" {
		want := map[string]bool{}
		for _, n := range strings.Split(only, ",") {
			want[strings.TrimSpace(n)] = true
		}
		var kept []Model
		for _, m := range models {
			if want[m.Name] {
				kept = append(kept, m)
			}
		}
		models = kept
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("no model: write %s, set VIVA_AIEVAL_BASE_URL, VIVA_AIEVAL_MODEL and VIVA_AIEVAL_KEY, or pass -model name=baseurl,keyenv", configPath)
	}
	return models, nil
}

// vivaSource is the nearest directory from the working directory up whose go.mod says module github.com/vivaui/viva.
func vivaSource(given string) (string, error) {
	dir := given
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = wd
	}
	dir, _ = filepath.Abs(dir)
	for {
		if data, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
			if first, _, _ := strings.Cut(string(data), "\n"); strings.TrimSpace(first) == "module github.com/vivaui/viva" {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir || given != "" {
			return "", fmt.Errorf("%s is not inside the Viva source; pass -viva <directory>", dir)
		}
		dir = parent
	}
}
