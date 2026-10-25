package linux

import "time"

// What the desktop says of the mouse and the caret, from its XSETTINGS. A desktop
// that publishes none keeps the values Viva always used.
const (
	defaultDoubleClick         = 500 * time.Millisecond
	defaultDoubleClickDistance = 4
	defaultBlinkInterval       = 500 * time.Millisecond
)

// clickTiming is how close in time and (in logical pixels) in space two presses
// must be to count as a double click.
func clickTiming(settings xsettings, scale float32) (time.Duration, float32) {
	interval, distance := defaultDoubleClick, float32(defaultDoubleClickDistance)
	if settings.hasDoubleClick && settings.DoubleClickTime > 0 {
		interval = time.Duration(min(settings.DoubleClickTime, 5000)) * time.Millisecond
	}
	if settings.hasDoubleClickDist && settings.DoubleClickDistance > 0 {
		// The desktop counts device pixels and a press is reported in logical ones.
		distance = float32(min(settings.DoubleClickDistance, 64)) / max(scale, 1)
	}
	return interval, distance
}

// caretBlink is whether the caret blinks and how long it stays in each state.
// Gtk/CursorBlinkTime is the whole cycle, on and off.
func caretBlink(settings xsettings) (blinks bool, interval time.Duration) {
	blinks, interval = true, defaultBlinkInterval
	if settings.hasCursorBlink && settings.CursorBlink == 0 {
		return false, interval
	}
	if settings.hasCursorBlinkTime && settings.CursorBlinkTime > 0 {
		interval = time.Duration(min(max(settings.CursorBlinkTime, 200), 4000)) * time.Millisecond / 2
	}
	return blinks, interval
}
