package leaks

import "testing"

func TestCountersReportWhatWasNotReleased(t *testing.T) {
	counter := New("test-resource")
	if len(Leaked()) != 0 {
		t.Fatalf("a fresh counter is leaking: %v", Leaked())
	}
	counter.Acquire()
	counter.Acquire()
	counter.Release()
	got := Leaked()
	if len(got) != 1 || got[0] != "test-resource=1" {
		t.Fatalf("Leaked = %v, want [test-resource=1]", got)
	}
	counter.Release()
	if len(Leaked()) != 0 {
		t.Fatalf("still leaking after the release: %v", Leaked())
	}
}

func TestBaselineSeesOnlyWhatChangedSinceItWasTaken(t *testing.T) {
	counter := New("baseline-resource")
	counter.Acquire() // held before the snapshot: not the test's to give back
	before := snapshot()
	if got := changedSince(before); len(got) != 0 {
		t.Fatalf("an unchanged counter was reported: %v", got)
	}
	counter.Acquire()
	if got := changedSince(before); len(got) != 1 {
		t.Fatalf("a resource acquired since the snapshot was not reported: %v", got)
	}
	counter.Release()
	counter.Release()
}
