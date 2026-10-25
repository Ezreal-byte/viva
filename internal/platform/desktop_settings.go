package platform

import "sync/atomic"

// DesktopSetting names a group of desktop-wide settings, besides the light or dark
// appearance, that an adapter reports when the user changes them.
type DesktopSetting uint8

const (
	// SettingFonts is how the desktop draws text: its interface font, antialiasing,
	// hinting or subpixel order. Whatever was measured or rasterized with the old
	// ones is stale.
	SettingFonts DesktopSetting = 1 << iota
	// SettingMotion is the preference for reduced animation.
	SettingMotion
)

var desktopObserver atomic.Pointer[func(DesktopSetting)]

// SetDesktopObserver sets who hears of a change in the desktop's settings. Nil stops it.
func SetDesktopObserver(observer func(DesktopSetting)) {
	if observer == nil {
		desktopObserver.Store(nil)
		return
	}
	desktopObserver.Store(&observer)
}

// NotifyDesktopChanged is for adapters: they call it on the event-loop thread when the
// settings in changed differ from what they last reported.
func NotifyDesktopChanged(changed DesktopSetting) {
	if observer := desktopObserver.Load(); observer != nil && changed != 0 {
		(*observer)(changed)
	}
}

var watchHook atomic.Pointer[func()]

// SetWatchHook is for adapters whose event loop sleeps until something wakes it: hook runs,
// on whatever goroutine changed the appearance observer, so that the adapter can wake its
// loop to look at DesktopWatched again. Nil removes it.
func SetWatchHook(hook func()) {
	if hook == nil {
		watchHook.Store(nil)
		return
	}
	watchHook.Store(&hook)
}

func watchChanged() {
	if hook := watchHook.Load(); hook != nil {
		(*hook)()
	}
}

// DesktopWatched reports whether anyone listens for a change in the system's appearance.
// An adapter that must keep a connection open to hear the desktop does so only while this
// is true, so an application that does not ask pays nothing; the settings an adapter hears
// of at no cost (a window's own property events) are reported to the desktop observer
// whether or not it is.
func DesktopWatched() bool {
	return appearanceObserver.Load() != nil
}
