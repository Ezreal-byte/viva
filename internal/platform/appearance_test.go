package platform

import "testing"

func TestAppearanceNotificationsReachTheObserverUntilItIsRemoved(t *testing.T) {
	NotifyAppearanceChanged() // nobody is listening: nothing happens
	calls := 0
	SetAppearanceObserver(func() { calls++ })
	NotifyAppearanceChanged()
	NotifyAppearanceChanged()
	SetAppearanceObserver(nil)
	NotifyAppearanceChanged()
	if calls != 2 {
		t.Fatalf("the observer was called %d times, want 2", calls)
	}
}
