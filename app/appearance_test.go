//go:build darwin || windows || linux

package app

import "testing"

func TestADriverAnswersForTheSystemAppearanceAndTellsWatchers(t *testing.T) {
	driver, err := NewDriver(Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()

	if got := SystemAppearance(); got != AppearanceUnknown {
		t.Fatalf("a headless run starts with %v, want unknown: it must not depend on the machine", got)
	}
	changes := 0
	stop := WatchSystemAppearance(func() { changes++ })
	driver.SetSystemAppearance(AppearanceDark)
	if got := SystemAppearance(); got != AppearanceDark || changes != 1 {
		t.Fatalf("after dark: %v, %d changes", got, changes)
	}
	driver.SetSystemAppearance(AppearanceDark)
	if changes != 1 {
		t.Fatalf("saying dark again counted as a change (%d)", changes)
	}
	driver.SetSystemAppearance(AppearanceLight)
	if got := SystemAppearance(); got != AppearanceLight || changes != 2 {
		t.Fatalf("after light: %v, %d changes", got, changes)
	}
	stop()
	stop() // idempotent
	driver.SetSystemAppearance(AppearanceDark)
	if changes != 2 {
		t.Fatalf("a stopped watcher heard %d changes", changes)
	}
}

func TestTwoWatchersBothHearAndEachStopsAlone(t *testing.T) {
	driver, err := NewDriver(Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	var first, second int
	stopFirst := WatchSystemAppearance(func() { first++ })
	stopSecond := WatchSystemAppearance(func() { second++ })
	defer stopSecond()
	driver.SetSystemAppearance(AppearanceDark)
	stopFirst()
	driver.SetSystemAppearance(AppearanceLight)
	if first != 1 || second != 2 {
		t.Fatalf("first heard %d, second heard %d; want 1 and 2", first, second)
	}
}

func TestDriverAppearanceResetsWhenTheLastDriverCloses(t *testing.T) {
	driver, err := NewDriver(Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	driver.SetSystemAppearance(AppearanceDark)
	driver.Close()
	driver.Close()
	again, err := NewDriver(Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	if got := SystemAppearance(); got != AppearanceUnknown {
		t.Fatalf("a new driver inherits %v", got)
	}
}

func TestAppearanceNames(t *testing.T) {
	for appearance, want := range map[Appearance]string{AppearanceUnknown: "unknown", AppearanceLight: "light", AppearanceDark: "dark"} {
		if appearance.String() != want {
			t.Fatalf("%d is %q, want %q", appearance, appearance.String(), want)
		}
	}
	if appearanceOf(true, false) != AppearanceUnknown || appearanceOf(true, true) != AppearanceDark || appearanceOf(false, true) != AppearanceLight {
		t.Fatal("appearanceOf maps wrongly")
	}
}

func TestADriverNotesWhatTheApplicationAsksOfTheNativeLookWithoutTouchingTheSystem(t *testing.T) {
	driver, err := NewDriver(Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if driver.NativeAppearance() != AppearanceUnknown {
		t.Fatal("a fresh driver was already asked for a look")
	}
	SetNativeAppearance(AppearanceDark)
	if driver.NativeAppearance() != AppearanceDark {
		t.Fatalf("native look = %v, want dark", driver.NativeAppearance())
	}
	if SystemAppearance() != AppearanceUnknown {
		t.Fatal("pinning the native look changed what the system says")
	}
	driver.Close()
	again, err := NewDriver(Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	if again.NativeAppearance() != AppearanceUnknown {
		t.Fatal("a new driver inherits the last one's request")
	}
}
