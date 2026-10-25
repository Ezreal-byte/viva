package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Stat aggregates the runs of one task for one model.
type Stat struct {
	Task, Model string
	Runs        int
	FirstTry    int // compiled in the first reply
	Compiled    int // compiled within the repair rounds
	Pass        int // compiled and met every expectation
	Prompt      int
	Completion  int
	Seconds     float64
}

func summarize(tasks []Task, models []Model, runs []Run) []Stat {
	var out []Stat
	for _, t := range tasks {
		for _, m := range models {
			s := Stat{Task: t.Name, Model: m.Name}
			for _, r := range runs {
				if r.Task != t.Name || r.Model != m.Name {
					continue
				}
				s.Runs++
				s.Prompt += r.Prompt
				s.Completion += r.Completion
				s.Seconds += r.Seconds
				if r.FirstTry {
					s.FirstTry++
				}
				if r.Compiled {
					s.Compiled++
				}
				if r.Pass {
					s.Pass++
				}
			}
			out = append(out, s)
		}
	}
	return out
}

func verdict(r Run) string {
	switch {
	case r.Error != "":
		return "error: " + firstLine(r.Error)
	case r.Pass && r.FirstTry:
		return "pass (compiled first try)"
	case r.Pass:
		return fmt.Sprintf("pass (repaired in %d)", r.RepairRounds)
	case r.Compiled:
		var bad []string
		for _, c := range r.Checks {
			if !c.OK {
				bad = append(bad, c.Name)
			}
		}
		return "compiled, failed: " + strings.Join(bad, "; ")
	}
	return "did not compile"
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return truncate(line, 160)
}

// renderTable prints the pass rates, then the failed expectations and the most common
// compiler complaints, which is where the fixes to the guide come from.
func renderTable(tasks []Task, models []Model, stats []Stat, runs []Run) string {
	var b strings.Builder
	fmt.Fprintf(&b, "| task | model | runs | first try | compiled | passed | tokens in/out per run | s per run |\n|---|---|---|---|---|---|---|---|\n")
	total := map[string]*Stat{}
	for _, s := range stats {
		if s.Runs == 0 {
			continue
		}
		name := s.Task
		for _, t := range tasks {
			if t.Name == s.Task && t.Needs != "" {
				name += " (needs " + t.Needs + ")"
			}
		}
		fmt.Fprintf(&b, "| %s | %s | %d | %d | %d | %d | %d / %d | %.0f |\n", name, s.Model, s.Runs, s.FirstTry, s.Compiled, s.Pass,
			s.Prompt/s.Runs, s.Completion/s.Runs, s.Seconds/float64(s.Runs))
		skip := false
		for _, t := range tasks {
			skip = skip || (t.Name == s.Task && t.Needs != "")
		}
		if skip {
			continue
		}
		t := total[s.Model]
		if t == nil {
			t = &Stat{Task: "all", Model: s.Model}
			total[s.Model] = t
		}
		t.Runs += s.Runs
		t.FirstTry += s.FirstTry
		t.Compiled += s.Compiled
		t.Pass += s.Pass
	}
	b.WriteString("\nTotals (tasks that need a missing capability excluded):\n\n")
	for _, m := range models {
		if t := total[m.Name]; t != nil {
			fmt.Fprintf(&b, "- %s: %d runs, first try %d%%, compiled %d%%, passed %d%%\n", m.Name, t.Runs,
				100*t.FirstTry/t.Runs, 100*t.Compiled/t.Runs, 100*t.Pass/t.Runs)
		}
	}

	failures := map[string]int{}
	for _, r := range runs {
		for _, c := range r.Checks {
			if !c.OK {
				failures[r.Task+": "+c.Name+" - "+truncate(c.Detail, 120)]++
			}
		}
	}
	appendCounts(&b, "\nFailed expectations:\n", failures, 25)

	errs := map[string]int{}
	for _, r := range runs {
		for _, a := range r.Attempts {
			for _, line := range strings.Split(a.Vet, "\n") {
				if n := normalizeError(line); n != "" {
					errs[n]++
				}
			}
		}
	}
	appendCounts(&b, "\nMost common compiler complaints:\n", errs, 25)
	return b.String()
}

var position = regexp.MustCompile(`^(vet: )?(\./)?[\w./-]+\.go:\d+:\d+: `)

func normalizeError(line string) string {
	if !position.MatchString(line) {
		return ""
	}
	return truncate(position.ReplaceAllString(line, ""), 140)
}

func appendCounts(b *strings.Builder, title string, counts map[string]int, limit int) {
	if len(counts) == 0 {
		return
	}
	type kv struct {
		k string
		n int
	}
	var all []kv
	for k, n := range counts {
		all = append(all, kv{k, n})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].n > all[j].n || all[i].n == all[j].n && all[i].k < all[j].k })
	b.WriteString(title + "\n")
	for i, e := range all {
		if i == limit {
			break
		}
		fmt.Fprintf(b, "- %dx %s\n", e.n, e.k)
	}
}

// reportFile prints the table of an earlier run from its results.json.
func reportFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var saved struct{ Runs []Run }
	if err := json.Unmarshal(data, &saved); err != nil {
		return err
	}
	var tasks []Task
	var models []Model
	seenTask, seenModel := map[string]bool{}, map[string]bool{}
	for _, r := range saved.Runs {
		if !seenTask[r.Task] {
			seenTask[r.Task] = true
			tasks = append(tasks, Task{Name: r.Task, Needs: r.Needs})
		}
		if !seenModel[r.Model] {
			seenModel[r.Model] = true
			models = append(models, Model{Name: r.Model})
		}
	}
	fmt.Print(renderTable(tasks, models, summarize(tasks, models, saved.Runs), saved.Runs))
	return nil
}
