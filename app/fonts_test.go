package app

import (
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestMonospaceFamiliesAreSortedUniqueAndIncludeTheSystemsOwn(t *testing.T) {
	families := MonospaceFamilies()
	if runtime.GOOS == "js" {
		if len(families) != 0 {
			t.Fatalf("a page listed fonts: %v", families)
		}
		return
	}
	if len(families) == 0 {
		t.Skipf("%s reports no fixed-pitch families here", runtime.GOOS)
	}
	if !slices.IsSortedFunc(families, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) }) {
		t.Fatalf("not sorted: %v", families)
	}
	seen := map[string]bool{}
	for _, name := range families {
		if name == "" || seen[name] || strings.ContainsAny(name, "\n\x00") || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "@") {
			t.Fatalf("bad entry %q in %v", name, families)
		}
		seen[name] = true
	}
	want := map[string]string{"darwin": "Menlo", "windows": "Consolas"}[runtime.GOOS]
	if want != "" && !seen[want] {
		t.Fatalf("%s is missing from %v", want, families)
	}
}
