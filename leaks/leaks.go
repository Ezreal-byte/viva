// Package leaks counts the long-lived resources Viva hands out so a test can assert
// that it gave every one of them back.
//
// A Counter is a name and an atomic number: Acquire when a resource is created,
// Release when it is destroyed. The cost is one atomic add per window or driver,
// not per frame, so counters are always on and need no build tag. The numbers
// matter only to tests: Main fails a package's test run if any counter is above
// zero when the tests end, and Baseline lets one test check its own resources.
package leaks

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
)

// Counter counts the live resources of one kind.
type Counter struct {
	name string
	live atomic.Int64
}

var (
	mu       sync.Mutex
	counters []*Counter
)

// New registers a counter. Call it from a package-level variable.
func New(name string) *Counter {
	counter := &Counter{name: name}
	mu.Lock()
	counters = append(counters, counter)
	mu.Unlock()
	return counter
}

// Acquire records that a resource was created.
func (counter *Counter) Acquire() { counter.live.Add(1) }

// Release records that a resource was destroyed.
func (counter *Counter) Release() { counter.live.Add(-1) }

// Live is how many resources of this kind exist now.
func (counter *Counter) Live() int64 { return counter.live.Load() }

// Name is what the counter counts.
func (counter *Counter) Name() string { return counter.name }

// Leaked lists, as "name=count", every counter that is not zero.
func Leaked() []string {
	mu.Lock()
	defer mu.Unlock()
	var out []string
	for _, counter := range counters {
		if n := counter.Live(); n != 0 {
			out = append(out, fmt.Sprintf("%s=%d", counter.name, n))
		}
	}
	return out
}

// Runner is what Main needs from testing.M: Run. It is an interface so this
// package, which every release build links, does not import testing (and with
// it flag, regexp and the rest of the test runtime).
type Runner interface {
	Run() int
}

// Main runs a package's tests and then checks that every counter is back at zero.
// Use it as the package's TestMain:
//
//	func TestMain(m *testing.M) { os.Exit(leaks.Main(m)) }
func Main(m Runner) int {
	code := m.Run()
	if code == 0 {
		if leaked := Leaked(); len(leaked) > 0 {
			fmt.Fprintf(os.Stderr, "leaks: resources still held when the tests ended: %v\n", leaked)
			return 1
		}
	}
	return code
}

// TB is what Baseline needs from testing.TB. It is an interface for the same
// reason as Runner: no test dependency in a release build.
type TB interface {
	Helper()
	Error(args ...any)
}

// Baseline records the counters now and returns a check that fails the test if any
// counter differs from that record when it runs. Call it first in a test and defer
// or t.Cleanup the result:
//
//	t.Cleanup(leaks.Baseline(t))
func Baseline(t TB) func() {
	t.Helper()
	before := snapshot()
	return func() {
		t.Helper()
		for _, difference := range changedSince(before) {
			t.Error(difference)
		}
	}
}

func snapshot() []int64 {
	mu.Lock()
	defer mu.Unlock()
	values := make([]int64, len(counters))
	for i, counter := range counters {
		values[i] = counter.Live()
	}
	return values
}

// changedSince describes every counter whose value differs from a snapshot.
func changedSince(before []int64) []string {
	mu.Lock()
	defer mu.Unlock()
	var out []string
	for i, counter := range counters {
		if i < len(before) && counter.Live() != before[i] {
			out = append(out, fmt.Sprintf("%s: %d live, %d when the test began", counter.name, counter.Live(), before[i]))
		}
	}
	return out
}
