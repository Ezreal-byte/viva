package linux

// Where a window or a panel may stand on a desktop with several monitors. The
// numbers are root-window coordinates in device pixels, y down, as RandR and
// the window manager's _NET_WORKAREA report them.

type pixelRect struct{ x, y, width, height int }

func (rect pixelRect) empty() bool { return rect.width <= 0 || rect.height <= 0 }

func (rect pixelRect) intersect(other pixelRect) pixelRect {
	left, top := max(rect.x, other.x), max(rect.y, other.y)
	right, bottom := min(rect.x+rect.width, other.x+other.width), min(rect.y+rect.height, other.y+other.height)
	if right <= left || bottom <= top {
		return pixelRect{}
	}
	return pixelRect{left, top, right - left, bottom - top}
}

func monitorRect(monitor monitorGeometry) pixelRect {
	return pixelRect{monitor.X, monitor.Y, monitor.Width, monitor.Height}
}

// usableArea is the part of a monitor windows may use: the monitor less what the
// desktop's panels reserve. The window manager reports one work area for the
// whole desktop, so on a monitor that holds none of its edges the work area
// does not tell anything and the monitor is all there is; an empty overlap
// (a work area on another monitor) says the same.
func usableArea(monitor pixelRect, work pixelRect, haveWork bool) pixelRect {
	if !haveWork {
		return monitor
	}
	if area := monitor.intersect(work); !area.empty() {
		return area
	}
	return monitor
}

// primaryMonitor is the monitor to fall back on: the primary one, else the first.
func primaryMonitor(monitors []monitorGeometry) (monitorGeometry, bool) {
	for _, monitor := range monitors {
		if monitor.Primary && monitor.Width > 0 && monitor.Height > 0 {
			return monitor, true
		}
	}
	for _, monitor := range monitors {
		if monitor.Width > 0 && monitor.Height > 0 {
			return monitor, true
		}
	}
	return monitorGeometry{}, false
}

// Enough of a window on a monitor for the person to find and drag it.
const (
	visibleWidth  = 96
	visibleHeight = 48
)

// visiblePosition keeps the top-left corner a window asked for when enough of
// it is on some monitor, and otherwise centers it on the primary monitor: a
// position remembered from a desktop whose monitor is gone would open the
// window where nobody can reach it. moved says it was changed.
func visiblePosition(monitors []monitorGeometry, x, y, width, height int) (px, py int, moved bool) {
	window := pixelRect{x, y, width, height}
	needWidth, needHeight := min(width, visibleWidth), min(height, visibleHeight)
	for _, monitor := range monitors {
		if shown := window.intersect(monitorRect(monitor)); shown.width >= needWidth && shown.height >= needHeight {
			return x, y, false
		}
	}
	primary, ok := primaryMonitor(monitors)
	if !ok {
		return x, y, false
	}
	return primary.X + max(0, (primary.Width-width)/2), primary.Y + max(0, (primary.Height-height)/2), true
}
