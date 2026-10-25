package app

// HighContrast reports whether the user turned on a high-contrast theme, which
// asks an application to draw in strong colors with clear borders rather than its
// own palette. Windows reports it; macOS and Linux do not yet, and report false.
// WatchSystemAppearance calls its watchers when it changes, as when the look
// switches between light and dark, so read it while painting.
func HighContrast() bool { return nativeHighContrast() }

// appearanceState is what WatchSystemAppearance compares to know that the look
// the system is set to changed: the light or dark look, and high contrast.
func appearanceState() uint32 {
	state := uint32(SystemAppearance())
	if HighContrast() {
		state |= 1 << 8
	}
	return state
}
