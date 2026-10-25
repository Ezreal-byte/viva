package input

import "testing"

func TestKeyTrackerPairsReleasesWithPresses(t *testing.T) {
	var tracker KeyTracker
	if repeat, tracked := tracker.Press(HeldKey{Code: 7, Key: KeyLeft}); repeat || !tracked {
		t.Fatalf("first press = repeat %v tracked %v", repeat, tracked)
	}
	if repeat, _ := tracker.Press(HeldKey{Code: 7, Key: KeyLeft}); !repeat {
		t.Fatal("a second press of a key that is down is an auto-repeat")
	}
	tracker.Press(HeldKey{Code: 9, Text: "!"})
	if tracker.Len() != 2 {
		t.Fatalf("Len = %d", tracker.Len())
	}
	// The release carries what the press carried, not what the key says now.
	key, ok := tracker.Release(9)
	if !ok || key.Text != "!" {
		t.Fatalf("release = %+v %v", key, ok)
	}
	if _, ok := tracker.Release(9); ok {
		t.Fatal("a key released twice must not be reported twice")
	}
	if _, ok := tracker.Release(1234); ok {
		t.Fatal("a release without a press is an orphan and must be dropped")
	}
}

func TestKeyTrackerTakeDrainsInPressOrder(t *testing.T) {
	var tracker KeyTracker
	for code := uint32(1); code <= 3; code++ {
		tracker.Press(HeldKey{Code: code})
	}
	tracker.Release(2)
	all := tracker.Take(nil)
	if len(all) != 2 || all[0].Code != 1 || all[1].Code != 3 || tracker.Len() != 0 {
		t.Fatalf("Take = %+v, left %d", all, tracker.Len())
	}
}

func TestKeyTrackerIgnoresUncodedKeysAndOverflow(t *testing.T) {
	var tracker KeyTracker
	if _, tracked := tracker.Press(HeldKey{}); tracked {
		t.Fatal("a key without a code cannot be matched with its release")
	}
	for code := uint32(1); code <= MaxHeldKeys; code++ {
		tracker.Press(HeldKey{Code: code})
	}
	if _, tracked := tracker.Press(HeldKey{Code: 99}); tracked {
		t.Fatal("a full tracker must not claim to follow another key")
	}
}

func BenchmarkKeyTrackerPressRelease(b *testing.B) {
	var tracker KeyTracker
	b.ReportAllocs()
	for b.Loop() {
		tracker.Press(HeldKey{Code: 4, Key: KeyLeft, Text: "x"})
		tracker.Release(4)
	}
}
