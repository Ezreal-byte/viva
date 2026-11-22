package windows

import (
	"unsafe"

	"github.com/vivaui/viva/internal/platform"
)

const (
	ninPopupOpen          = 0x0406
	statusHoverTimer      = 1
	statusHoverPanelTimer = 2
	statusHoverDelay      = 400
	statusPanelInterval   = 90
)

type statusHover struct {
	pendingID uint32
	pointer   point
	panelID   uint32
	panel     *nativeWindow
	anchor    platform.ScreenRect
}

func (item *statusItem) SetHoverPanel(open func(platform.ScreenRect) platform.Window) bool {
	if item.removed.Load() {
		return false
	}
	previous := item.hoverPanel
	item.hoverPanel = open
	if (previous == nil) == (open == nil) {
		return true
	}
	if open == nil {
		item.application.status.stopItemHover(item.id)
	}
	if err := item.notify(notifyModify); err != nil && open != nil {
		// Let the caller keep its tooltip or click fallback when the shell
		// has not accepted the panel's notification flags.
		item.hoverPanel = previous
		return false
	}
	// TaskbarCreated reapplies the desired flags if Explorer is restarting.
	return true
}

func (item *statusItem) canHover() bool {
	return !item.removed.Load() && !item.menuOpen && (item.hoverPanel != nil || item.hoverMenu)
}

func (status *statusState) beginHover(item *statusItem) {
	if !item.canHover() || status.hover.panelID == item.id {
		return
	}
	var cursor point
	if ok, _, _ := getCursorPos.Call(uintptr(unsafe.Pointer(&cursor))); ok == 0 {
		return
	}
	status.hover.pointer = cursor
	if status.hover.pendingID != item.id {
		// Further mouse moves update the overflow anchor without postponing
		// the dwell deadline or waiting on another Explorer notification.
		status.hover.pendingID = item.id
		setTimer.Call(status.window, statusHoverTimer, statusHoverDelay, 0)
	}
}

func (status *statusState) cancelHover() {
	if status.hover.pendingID != 0 {
		killTimer.Call(status.window, statusHoverTimer)
		status.hover.pendingID = 0
	}
}

func (status *statusState) finishHover() {
	item := status.items[status.hover.pendingID]
	status.cancelHover()
	if item == nil || !item.canHover() {
		return
	}
	var cursor point
	if ok, _, _ := getCursorPos.Call(uintptr(unsafe.Pointer(&cursor))); ok == 0 {
		return
	}
	bounds := hoverAnchor(item.anchor(status.window, 0), status.hover.pointer)
	if hoverContains(bounds, cursor) {
		status.openHover(item, bounds)
	}
}

func (status *statusState) openHover(item *statusItem, anchor platform.ScreenRect) {
	if !item.canHover() || status.hover.panelID == item.id {
		return
	}
	status.closeHoverPanel()
	if item.hoverPanel == nil {
		item.popMenu(status.window, anchor)
		return
	}
	panel, ok := item.hoverPanel(anchor).(*nativeWindow)
	if !ok || panel == nil {
		return
	}
	status.hover.panelID, status.hover.panel, status.hover.anchor = item.id, panel, anchor
	setTimer.Call(status.window, statusHoverPanelTimer, statusPanelInterval, 0)
}

func (status *statusState) checkHoverPanel() {
	panel := status.hover.panel
	if panel == nil || panel.closed.Load() {
		status.closeHoverPanel()
		return
	}
	var cursor point
	var bounds rect
	if haveCursor, _, _ := getCursorPos.Call(uintptr(unsafe.Pointer(&cursor))); haveCursor == 0 {
		return
	}
	if haveBounds, _, _ := getWindowRect.Call(panel.state.window, uintptr(unsafe.Pointer(&bounds))); haveBounds == 0 ||
		!hoverContains(hoverPanelRegion(status.hover.anchor, bounds), cursor) {
		status.closeHoverPanel()
	}
}

func (status *statusState) closeHoverPanel() {
	if panel := status.hover.panel; panel != nil {
		killTimer.Call(status.window, statusHoverPanelTimer)
		status.hover.panelID, status.hover.panel = 0, nil
		panel.Close()
	}
}

func (status *statusState) stopItemHover(id uint32) {
	if status.hover.pendingID == id {
		status.cancelHover()
	}
	if status.hover.panelID == id {
		status.closeHoverPanel()
	}
}

func hoverContains(bounds platform.ScreenRect, cursor point) bool {
	x, y := float32(cursor.X), float32(cursor.Y)
	return x >= bounds.X && x < bounds.X+bounds.Width && y >= bounds.Y && y < bounds.Y+bounds.Height
}

func hoverAnchor(icon platform.ScreenRect, lastMove point) platform.ScreenRect {
	if hoverContains(icon, lastMove) {
		return icon
	}
	// Explorer can report a different rectangle for overflow icons. The
	// last WM_MOUSEMOVE from this icon is an anchor in the visible flyout.
	return platform.ScreenRect{X: float32(lastMove.X) - 8, Y: float32(lastMove.Y) - 8, Width: 16, Height: 16}
}

func hoverPanelRegion(icon platform.ScreenRect, panel rect) platform.ScreenRect {
	// Include the gap between icon and panel to let the pointer cross it.
	x, y := min(icon.X, float32(panel.Left)), min(icon.Y, float32(panel.Top))
	return platform.ScreenRect{
		X: x, Y: y,
		Width:  max(icon.X+max(1, icon.Width), float32(panel.Right)) - x,
		Height: max(icon.Y+max(1, icon.Height), float32(panel.Bottom)) - y,
	}
}
