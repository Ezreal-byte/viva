package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "record":
		err = record(os.Args[2:])
	case "check":
		err = check(os.Args[2:])
	case "steps":
		err = steps(os.Args[2:])
	case "compare":
		err = compare(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "perfhistory:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  perfhistory record -machine LABEL -kind physical|vm|software-gpu [-binary PATH] [-dir DIR] < go-test-bench-output
  perfhistory check  -machine LABEL -kind KIND [-threshold 0.15] [-window 5] [-dir DIR] < go-test-bench-output
  perfhistory steps  -machine LABEL -kind KIND -cpu CPU [-width 3] [-threshold 0.25] [-dir DIR]
  perfhistory compare [-threshold 0.15] [-min-ns 20] BASE-OUTPUT < CANDIDATE-OUTPUT`)
	os.Exit(2)
}

const defaultDir = "docs/benchmarks/history"

func commonFlags(name string) (*flag.FlagSet, *string, *string, *string) {
	flags := flag.NewFlagSet(name, flag.ExitOnError)
	label := flags.String("machine", "", "a label for this machine, such as m2-pro")
	kind := flags.String("kind", "", "physical, vm or software-gpu: results of different kinds are never compared")
	dir := flags.String("dir", defaultDir, "directory of history files")
	return flags, label, kind, dir
}

func machineOf(parsed Machine, label, kind string) (Machine, error) {
	if label == "" || kind == "" {
		return Machine{}, fmt.Errorf("-machine and -kind are required: a result means nothing without the machine it came from")
	}
	if kind != "physical" && kind != "vm" && kind != "software-gpu" {
		return Machine{}, fmt.Errorf("-kind %q is not physical, vm or software-gpu", kind)
	}
	parsed.Label, parsed.Kind = label, kind
	if parsed.GOOS == "" {
		parsed.GOOS = runtime.GOOS
	}
	if parsed.GOARCH == "" {
		parsed.GOARCH = runtime.GOARCH
	}
	return parsed, nil
}

func record(args []string) error {
	flags, label, kind, dir := commonFlags("record")
	binary := flags.String("binary", "", "a stripped application binary whose size to record")
	flags.Parse(args)
	results, parsed, err := parseBenchmarks(os.Stdin)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("no benchmark results on standard input")
	}
	machine, err := machineOf(parsed, *label, *kind)
	if err != nil {
		return err
	}
	run := Run{
		Time: time.Now().UTC().Format(time.RFC3339), GoVersion: runtime.Version(), Machine: machine, Results: results,
	}
	run.Commit, run.Dirty = gitState()
	if *binary != "" {
		info, err := os.Stat(*binary)
		if err != nil {
			return err
		}
		run.BinaryBytes = info.Size()
	}
	path := historyPath(*dir, *label)
	if err := appendRun(path, run); err != nil {
		return err
	}
	fmt.Printf("recorded %d benchmarks to %s\n", len(results), path)
	return nil
}

func check(args []string) error {
	flags, label, kind, dir := commonFlags("check")
	threshold := flags.Float64("threshold", 0.15, "fractional slowdown that is a finding")
	window := flags.Int("window", 5, "how many earlier runs make the baseline")
	flags.Parse(args)
	results, parsed, err := parseBenchmarks(os.Stdin)
	if err != nil {
		return err
	}
	machine, err := machineOf(parsed, *label, *kind)
	if err != nil {
		return err
	}
	history, err := readHistory(historyPath(*dir, *label))
	if err != nil {
		return err
	}
	findings := checkRun(history, Run{Machine: machine, Results: results}, *threshold, *window)
	compared := 0
	for _, result := range results {
		if len(seriesOf(history, machine, result.Name).ns) > 0 {
			compared++
		}
	}
	fmt.Printf("%d of %d benchmarks have comparable history on %s\n", compared, len(results), *label)
	for _, finding := range findings {
		fmt.Printf("REGRESSION %s: %s\n", finding.Name, finding.Detail)
	}
	if len(findings) > 0 {
		return fmt.Errorf("%d regressions", len(findings))
	}
	return nil
}

// compare checks one benchmark run against another taken on the same machine,
// with the same threshold rules as check: no history file is involved, so it can
// guard a pull request by comparing the base revision with the candidate.
func compare(args []string) error {
	flags := flag.NewFlagSet("compare", flag.ExitOnError)
	threshold := flags.Float64("threshold", 0.15, "fractional slowdown that is a finding")
	minimumNs := flags.Float64("min-ns", 20, "least absolute slowdown a finding needs, in nanoseconds")
	flags.Parse(args)
	if flags.NArg() != 1 {
		return fmt.Errorf("compare takes one file with the base benchmark output; the candidate run is on standard input")
	}
	file, err := os.Open(flags.Arg(0))
	if err != nil {
		return err
	}
	defer file.Close()
	base, baseMachine, err := parseBenchmarks(file)
	if err != nil {
		return err
	}
	candidate, candidateMachine, err := parseBenchmarks(os.Stdin)
	if err != nil {
		return err
	}
	if len(base) == 0 || len(candidate) == 0 {
		return fmt.Errorf("no benchmark results in the base or the candidate output")
	}
	findings, err := compareRuns(base, baseMachine, candidate, candidateMachine, *threshold, *minimumNs)
	if err != nil {
		return err
	}
	compared := 0
	for _, result := range candidate {
		for _, earlier := range base {
			if earlier.Name == result.Name {
				compared++
				break
			}
		}
	}
	fmt.Printf("%d of %d benchmarks have a base result\n", compared, len(candidate))
	for _, finding := range findings {
		fmt.Printf("REGRESSION %s: %s\n", finding.Name, finding.Detail)
	}
	if len(findings) > 0 {
		return fmt.Errorf("%d regressions", len(findings))
	}
	return nil
}

func steps(args []string) error {
	flags, label, kind, dir := commonFlags("steps")
	cpu := flags.String("cpu", "", "the CPU line of the runs to read (they must match exactly)")
	width := flags.Int("width", 3, "runs on each side of a step")
	threshold := flags.Float64("threshold", 0.25, "fractional change that is a step")
	flags.Parse(args)
	history, err := readHistory(historyPath(*dir, *label))
	if err != nil {
		return err
	}
	if len(history) == 0 {
		return fmt.Errorf("no history for %s in %s", *label, *dir)
	}
	machine := history[len(history)-1].Machine
	machine.Kind = *kind
	if *cpu != "" {
		machine.CPU = *cpu
	}
	names := map[string]struct{}{}
	for _, run := range history {
		if comparable(run.Machine, machine) {
			for _, result := range run.Results {
				names[result.Name] = struct{}{}
			}
		}
	}
	found := 0
	for name := range names {
		for _, step := range stepsOf(history, machine, name, *width, *threshold) {
			found++
			fmt.Printf("STEP %s at run %d (%s): x%.2f\n", step.Name, step.Index, step.Time, step.Ratio)
		}
	}
	fmt.Printf("%d steps in %d benchmarks\n", found, len(names))
	return nil
}

func gitState() (commit string, dirty bool) {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err == nil {
		commit = strings.TrimSpace(string(out))
	}
	status, err := exec.Command("git", "status", "--porcelain").Output()
	return commit, err == nil && len(strings.TrimSpace(string(status))) > 0
}
