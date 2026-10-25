package app

import "github.com/vivaui/viva/leaks"

// What the application layer must give back. The tests of this package (and of the
// ones that drive it) fail if any is still held when they end.
var (
	liveDrivers          = leaks.New("app drivers")
	liveDriverAppearance = leaks.New("app driver appearance holds")
	liveWindows          = leaks.New("app windows")
)
