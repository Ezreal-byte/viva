package platform

// WindowControlWidth is the width in logical pixels of each of the minimize,
// maximize and close buttons Viva draws at the right of a custom titlebar, and
// where the maximize button lies: second from the right.
const WindowControlWidth = 46

// MaximizeButtonAt reports whether a point of a custom-titlebar window of the
// given width lies on the maximize button, which is strip tall. Windows 11 shows
// its snap layouts over a button the window calls its maximize button, so the
// adapter needs to know where Viva drew it.
func MaximizeButtonAt(x, y, width, strip float32) bool {
	left := width - 2*WindowControlWidth
	return y >= 0 && y < strip && x >= left && x < left+WindowControlWidth
}

// Edges of the screen an auto-hiding bar can be on, in the order of ABE_LEFT,
// ABE_TOP, ABE_RIGHT and ABE_BOTTOM.
const (
	EdgeIndexLeft = iota
	EdgeIndexTop
	EdgeIndexRight
	EdgeIndexBottom
)

// MaximizedClient is the client area of a maximized window with no frame of its
// own. Windows proposes a rectangle that overhangs the monitor by the frame; the
// client is pulled back in by horizontal and vertical. An auto-hiding taskbar
// reveals itself when the pointer reaches the screen edge it is on, which it
// cannot do over a window that covers that edge to the last pixel, so one pixel
// of that edge is left uncovered.
func MaximizedClient(left, top, right, bottom, horizontal, vertical int32, autoHide [4]bool) (int32, int32, int32, int32) {
	left, top, right, bottom = left+horizontal, top+vertical, right-horizontal, bottom-vertical
	if autoHide[EdgeIndexLeft] {
		left++
	}
	if autoHide[EdgeIndexTop] {
		top++
	}
	if autoHide[EdgeIndexRight] {
		right--
	}
	if autoHide[EdgeIndexBottom] {
		bottom--
	}
	return left, top, right, bottom
}
