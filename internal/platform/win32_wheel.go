package platform

// WheelNotch is the delta of one click of a classic mouse wheel (WHEEL_DELTA).
const WheelNotch = 120

// WheelPageScroll is the system's "one screen per notch" setting (WHEEL_PAGESCROLL).
const WheelPageScroll = ^uint32(0)

// wheelLinePixels is the height in logical pixels of one line a notch scrolls.
const wheelLinePixels = 16

// WheelPixels is the scroll in logical pixels of a wheel message that reports
// delta, with units per notch from the user's setting (lines for the vertical
// wheel, characters for the horizontal one), or WheelPageScroll for a whole page
// of the given height. A delta that is not a whole number of notches comes from
// a precision touchpad or a free-spinning wheel, which report the distance they
// moved: it is scaled in proportion and reported as precise, so the application
// moves by exactly that and not by at least a line.
func WheelPixels(delta int, units uint32, page float32) (pixels float32, precise bool) {
	perNotch := page
	if units != WheelPageScroll {
		perNotch = float32(units) * wheelLinePixels
	}
	return float32(delta) * perNotch / WheelNotch, delta%WheelNotch != 0
}
