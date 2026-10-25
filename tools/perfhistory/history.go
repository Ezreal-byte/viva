// Command perfhistory keeps a time series of benchmark results, one file per
// machine, and says when a result steps away from its own history.
//
//	sh benchmarks/m2_core.sh | go run ./tools/perfhistory record -machine m2-pro -kind physical
//	go run ./tools/perfhistory check -machine m2-pro < new-run.txt
//	go run ./tools/perfhistory steps -machine m2-pro
//
// A result is only ever compared with results from the same machine label, CPU,
// operating system and kind of environment (physical hardware, virtual machine,
// software GPU), because a number from one is not a number from another.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Machine is what a result was measured on. Two runs are comparable only if every
// field is equal.
type Machine struct {
	Label  string `json:"label"`
	Kind   string `json:"kind"` // physical, vm or software-gpu
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	CPU    string `json:"cpu"`
}

// Metric is a benchmark's own unit, such as "items" or "bytes/host".
type Metric struct {
	Unit  string  `json:"unit"`
	Value float64 `json:"value"`
}

// Result is one benchmark in one run. With -count above one the run's lines are
// merged: the median time, the largest allocation count.
type Result struct {
	Name        string   `json:"name"`
	NsPerOp     float64  `json:"ns_per_op"`
	BytesPerOp  float64  `json:"bytes_per_op"`
	AllocsPerOp float64  `json:"allocs_per_op"`
	Samples     int      `json:"samples"`
	Extra       []Metric `json:"extra,omitempty"`
}

// Run is everything recorded by one invocation of record.
type Run struct {
	Time        string   `json:"time"`
	Commit      string   `json:"commit"`
	Dirty       bool     `json:"dirty"`
	GoVersion   string   `json:"go_version"`
	Machine     Machine  `json:"machine"`
	BinaryBytes int64    `json:"binary_bytes,omitempty"`
	Results     []Result `json:"results"`
}

// parseBenchmarks reads `go test -bench` output. It returns the results, merged
// by name, and the cpu, goos and goarch lines the tool prints, which identify the
// machine.
func parseBenchmarks(reader io.Reader) (results []Result, machine Machine, err error) {
	type sample struct {
		ns, bytes, allocs float64
		extra             []Metric
	}
	samples := map[string][]sample{}
	var order []string
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "goos: "):
			machine.GOOS = strings.TrimPrefix(line, "goos: ")
		case strings.HasPrefix(line, "goarch: "):
			machine.GOARCH = strings.TrimPrefix(line, "goarch: ")
		case strings.HasPrefix(line, "cpu: "):
			machine.CPU = strings.TrimPrefix(line, "cpu: ")
		case strings.HasPrefix(line, "Benchmark"):
			name, current, ok := parseBenchmarkLine(line)
			if !ok {
				continue
			}
			if _, seen := samples[name]; !seen {
				order = append(order, name)
			}
			samples[name] = append(samples[name], sample{ns: current.NsPerOp, bytes: current.BytesPerOp, allocs: current.AllocsPerOp, extra: current.Extra})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, machine, err
	}
	for _, name := range order {
		all := samples[name]
		times := make([]float64, len(all))
		result := Result{Name: name, Samples: len(all), Extra: all[len(all)-1].extra}
		for i, s := range all {
			times[i] = s.ns
			result.BytesPerOp = max(result.BytesPerOp, s.bytes)
			result.AllocsPerOp = max(result.AllocsPerOp, s.allocs)
		}
		result.NsPerOp = median(times)
		results = append(results, result)
	}
	return results, machine, nil
}

// parseBenchmarkLine reads "BenchmarkX-10  1000  1368 ns/op  0 B/op  0 allocs/op  7 items".
func parseBenchmarkLine(line string) (string, Result, bool) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return "", Result{}, false
	}
	name := fields[0]
	if cut := strings.LastIndexByte(name, '-'); cut > 0 {
		if _, err := strconv.Atoi(name[cut+1:]); err == nil {
			name = name[:cut] // the GOMAXPROCS suffix belongs to the machine, not the benchmark
		}
	}
	if _, err := strconv.Atoi(fields[1]); err != nil {
		return "", Result{}, false
	}
	result := Result{Name: name}
	found := false
	for i := 2; i+1 < len(fields); i += 2 {
		value, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			continue
		}
		switch unit := fields[i+1]; unit {
		case "ns/op":
			result.NsPerOp, found = value, true
		case "B/op":
			result.BytesPerOp = value
		case "allocs/op":
			result.AllocsPerOp = value
		default:
			result.Extra = append(result.Extra, Metric{Unit: unit, Value: value})
		}
	}
	return name, result, found
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	middle := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[middle]
	}
	return (sorted[middle-1] + sorted[middle]) / 2
}

// comparable reports whether results measured on a and b may be compared.
func comparable(a, b Machine) bool { return a == b }

// path is the history file of a machine label inside dir.
func historyPath(dir, label string) string {
	return filepath.Join(dir, safeName(label)+".jsonl")
}

func safeName(label string) string {
	var out strings.Builder
	for _, r := range label {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			out.WriteRune(r)
		default:
			out.WriteByte('-')
		}
	}
	return out.String()
}

// readHistory returns every run in a history file, oldest first. A missing file is
// an empty history.
func readHistory(path string) ([]Run, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var runs []Run
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 8<<20)
	for line := 1; scanner.Scan(); line++ {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var run Run
		if err := json.Unmarshal(scanner.Bytes(), &run); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, line, err)
		}
		runs = append(runs, run)
	}
	return runs, scanner.Err()
}

func appendRun(path string, run Run) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(run)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// series is the values one benchmark took across the comparable runs, oldest first.
type series struct {
	ns     []float64
	allocs []float64
}

func seriesOf(runs []Run, machine Machine, name string) series {
	var out series
	for _, run := range runs {
		if !comparable(run.Machine, machine) {
			continue
		}
		for _, result := range run.Results {
			if result.Name == name {
				out.ns = append(out.ns, result.NsPerOp)
				out.allocs = append(out.allocs, result.AllocsPerOp)
			}
		}
	}
	return out
}

// Finding is one result that moved away from its history.
type Finding struct {
	Name   string
	Detail string
}

// minimumSlowdownNs is the least absolute slowdown a finding needs. A few
// nanoseconds is the resolution of the clock and the noise of a shared machine, so
// 3 ns becoming 4 ns is a 33% change that means nothing.
const minimumSlowdownNs = 20

// checkRun compares a new run with the comparable history: a time more than
// threshold above the median of the last window runs (and more than
// minimumSlowdownNs above it), or any increase in allocations, is a finding. A
// benchmark with no history is not.
func checkRun(history []Run, candidate Run, threshold float64, window int) []Finding {
	var findings []Finding
	for _, result := range candidate.Results {
		past := seriesOf(history, candidate.Machine, result.Name)
		if len(past.ns) == 0 {
			continue
		}
		recent := past.ns[max(0, len(past.ns)-window):]
		base := median(recent)
		if base > 0 && result.NsPerOp > base*(1+threshold) && result.NsPerOp-base >= minimumSlowdownNs {
			findings = append(findings, Finding{result.Name, fmt.Sprintf("%.0f ns/op, %.0f%% above the median %.0f of the last %d runs", result.NsPerOp, (result.NsPerOp/base-1)*100, base, len(recent))})
		}
		lastAllocs := past.allocs[len(past.allocs)-1]
		if result.AllocsPerOp > lastAllocs {
			findings = append(findings, Finding{result.Name, fmt.Sprintf("%.0f allocs/op, was %.0f", result.AllocsPerOp, lastAllocs)})
		}
	}
	return findings
}

// Step is a lasting change in a series: the median of the runs after a point is
// ratio times the median of the runs before it.
type Step struct {
	Name  string
	Index int
	Ratio float64
	Time  string
}

// stepsOf scans a benchmark's series for lasting changes, not only the last run's:
// at each point the median of the next width values against the median of the
// previous width values.
func stepsOf(runs []Run, machine Machine, name string, width int, threshold float64) []Step {
	var times []string
	var values []float64
	for _, run := range runs {
		if !comparable(run.Machine, machine) {
			continue
		}
		for _, result := range run.Results {
			if result.Name == name {
				times = append(times, run.Time)
				values = append(values, result.NsPerOp)
			}
		}
	}
	var steps []Step
	last := -width
	for i := width; i+width <= len(values); i++ {
		before, after := median(values[i-width:i]), median(values[i:i+width])
		if before <= 0 {
			continue
		}
		ratio := after / before
		if math.Abs(ratio-1) >= threshold && i-last >= width {
			steps = append(steps, Step{Name: name, Index: i, Ratio: ratio, Time: times[i]})
			last = i
		}
	}
	return steps
}
