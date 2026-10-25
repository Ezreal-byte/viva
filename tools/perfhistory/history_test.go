package main

import (
	"path/filepath"
	"strings"
	"testing"
)

const sampleOutput = `goos: darwin
goarch: arm64
pkg: github.com/vivaui/viva/app
cpu: Apple M2 Pro
BenchmarkClick-10    	  800000	      1400 ns/op	       0 B/op	       0 allocs/op
BenchmarkClick-10    	  800000	      1300 ns/op	       0 B/op	       0 allocs/op
BenchmarkClick-10    	  800000	      9999 ns/op	       0 B/op	       0 allocs/op
BenchmarkWhole/hosts-10-10   	     200	    201293 ns/op	       167.0 items	     160 B/op	       5 allocs/op
PASS
`

func TestParsingMergesRepeatedRunsByMedianAndKeepsTheMachine(t *testing.T) {
	results, machine, err := parseBenchmarks(strings.NewReader(sampleOutput))
	if err != nil {
		t.Fatal(err)
	}
	if machine.GOOS != "darwin" || machine.GOARCH != "arm64" || machine.CPU != "Apple M2 Pro" {
		t.Fatalf("machine = %+v", machine)
	}
	if len(results) != 2 {
		t.Fatalf("results = %+v", results)
	}
	click := results[0]
	if click.Name != "BenchmarkClick" || click.NsPerOp != 1400 || click.Samples != 3 {
		t.Fatalf("click = %+v; the median of three runs must beat the one outlier", click)
	}
	whole := results[1]
	if whole.Name != "BenchmarkWhole/hosts-10" || whole.AllocsPerOp != 5 || len(whole.Extra) != 1 || whole.Extra[0].Unit != "items" {
		t.Fatalf("whole = %+v; the -10 of the sub-benchmark name belongs to it, only the last suffix is GOMAXPROCS", whole)
	}
}

func run(machine Machine, ns, allocs float64) Run {
	return Run{Machine: machine, Time: "t", Results: []Result{{Name: "BenchmarkX", NsPerOp: ns, AllocsPerOp: allocs}}}
}

func TestResultsFromDifferentMachinesAreNeverCompared(t *testing.T) {
	laptop := Machine{Label: "laptop", Kind: "physical", GOOS: "darwin", GOARCH: "arm64", CPU: "M2"}
	vm := laptop
	vm.Kind = "vm"
	history := []Run{run(laptop, 100, 0), run(laptop, 100, 0), run(laptop, 100, 0)}
	if got := checkRun(history, run(vm, 100000, 9), 0.15, 5); len(got) != 0 {
		t.Fatalf("a VM result was compared with physical history: %v", got)
	}
	if got := checkRun(history, run(laptop, 100000, 0), 0.15, 5); len(got) != 1 {
		t.Fatalf("a real regression on the same machine was missed: %v", got)
	}
}

func TestCheckFlagsSlowdownsAboveTheThresholdAndAnyNewAllocation(t *testing.T) {
	machine := Machine{Label: "m", Kind: "physical"}
	history := []Run{run(machine, 100, 0), run(machine, 104, 0), run(machine, 98, 0)}
	if got := checkRun(history, run(machine, 110, 0), 0.15, 5); len(got) != 0 {
		t.Fatalf("a 10%% change flagged at a 15%% threshold: %v", got)
	}
	if got := checkRun(history, run(machine, 120, 0), 0.15, 5); len(got) != 1 {
		t.Fatalf("a 20%% slowdown not flagged: %v", got)
	}
	if got := checkRun(history, run(machine, 100, 1), 0.15, 5); len(got) != 1 || !strings.Contains(got[0].Detail, "allocs") {
		t.Fatalf("a new allocation not flagged: %v", got)
	}
	if got := checkRun(nil, run(machine, 1e9, 99), 0.15, 5); len(got) != 0 {
		t.Fatalf("a benchmark with no history was flagged: %v", got)
	}
}

func TestNanosecondNoiseOnTinyBenchmarksIsNotARegression(t *testing.T) {
	machine := Machine{Label: "m", Kind: "physical"}
	history := []Run{run(machine, 3, 0), run(machine, 3, 0), run(machine, 3, 0)}
	if got := checkRun(history, run(machine, 4.5, 0), 0.15, 5); len(got) != 0 {
		t.Fatalf("1.5 ns of noise on a 3 ns benchmark was flagged: %v", got)
	}
	if got := checkRun(history, run(machine, 60, 0), 0.15, 5); len(got) != 1 {
		t.Fatalf("a 20x slowdown of a tiny benchmark was ignored: %v", got)
	}
}

func TestAStepIsFoundWhereverItHappensNotOnlyAtTheLastRun(t *testing.T) {
	machine := Machine{Label: "m", Kind: "physical"}
	var runs []Run
	for _, ns := range []float64{100, 101, 99, 100, 160, 158, 161, 159, 160, 162} {
		runs = append(runs, run(machine, ns, 0))
	}
	steps := stepsOf(runs, machine, "BenchmarkX", 3, 0.25)
	if len(steps) != 1 || steps[0].Ratio < 1.4 {
		t.Fatalf("steps = %+v, want one step of about x1.6 in the middle of the series", steps)
	}
	flat := []Run{run(machine, 100, 0), run(machine, 103, 0), run(machine, 97, 0), run(machine, 101, 0), run(machine, 99, 0), run(machine, 102, 0)}
	if got := stepsOf(flat, machine, "BenchmarkX", 3, 0.25); len(got) != 0 {
		t.Fatalf("noise reported as a step: %v", got)
	}
}

func TestHistoryRoundTripsThroughItsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history", "m.jsonl")
	machine := Machine{Label: "m", Kind: "vm", GOOS: "linux", GOARCH: "amd64", CPU: "x"}
	for _, ns := range []float64{1, 2} {
		if err := appendRun(path, run(machine, ns, 0)); err != nil {
			t.Fatal(err)
		}
	}
	runs, err := readHistory(path)
	if err != nil || len(runs) != 2 || runs[1].Results[0].NsPerOp != 2 || runs[0].Machine != machine {
		t.Fatalf("runs = %+v, err = %v", runs, err)
	}
	if missing, err := readHistory(filepath.Join(t.TempDir(), "none.jsonl")); err != nil || missing != nil {
		t.Fatalf("a missing history = %v, %v; want empty", missing, err)
	}
}
