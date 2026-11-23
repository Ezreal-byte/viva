package windows

import (
	"syscall"
	"time"
	"unsafe"

	"github.com/vivaui/viva/internal/platform"
)

const (
	ninPopupOpen        = 0x0406
	ninPopupClose       = 0x0407
	statusHoverTimer    = 1
	statusHoverGapTimer = 2
	statusHoverDelay    = 400
	statusHoverGap      = 150

	wmStatusHoverLeave       = 0x8014
	wmStatusHoverDismissMenu = 0x8015
	wmStatusProbeIcon        = 0x8016
)

var (
	statusMouseHookCallback = syscall.NewCallback(statusMouseHook)
	endStatusMenu           = user.NewProc("EndMenu")
	statusWindowClassName   = user.NewProc("GetClassNameW")
	statusWindowThread      = user.NewProc("GetWindowThreadProcessId")
	statusAncestor          = user.NewProc("GetAncestor")
)

type statusHover struct {
	iconID       uint32
	anchor       platform.ScreenRect
	opened       bool
	dwellUntil   int64
	postedLeave  bool
	probePending bool
	watchEntry   bool
	hook         uintptr
	menuID       uint32
	panelID      uint32
	panel        *nativeWindow
	gapUntil     int64
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
	item.application.status.updateEntryWatch()
	// TaskbarCreated reapplies the desired flags if Explorer is restarting.
	return true
}

func (item *statusItem) canHover() bool {
	return !item.removed.Load() && !item.menuOpen && (item.hoverPanel != nil || item.hoverMenu)
}

func (status *statusState) beginHover(item *statusItem, cursor point, immediate bool) {
	h := &status.hover
	if !item.canHover() || h.menuID != 0 {
		return
	}
	if h.iconID != item.id {
		status.leaveHover()
		h.iconID = item.id
		h.anchor = hoverAnchor(item.anchor(status.window, 0), cursor)
		status.syncHoverMouseHook()
	}
	if h.panelID == item.id {
		status.cancelHoverGap()
	}
	if h.opened {
		return
	}
	if immediate {
		status.openHover()
	} else if h.dwellUntil == 0 {
		h.dwellUntil = monotonicNow() + statusHoverDelay*int64(time.Millisecond)
		setTimer.Call(status.window, statusHoverTimer, statusHoverDelay, 0)
	}
}

func (status *statusState) cancelHoverDwell() {
	if status.hover.dwellUntil != 0 {
		killTimer.Call(status.window, statusHoverTimer)
		status.hover.dwellUntil = 0
	}
}

func (status *statusState) finishHover() {
	h := &status.hover
	if h.dwellUntil == 0 || monotonicNow() < h.dwellUntil {
		return
	}
	var cursor point
	if ok, _, _ := getCursorPos.Call(uintptr(unsafe.Pointer(&cursor))); ok != 0 && hoverContains(h.anchor, cursor) {
		status.openHover()
	} else {
		status.leaveHover()
	}
}

func (status *statusState) openHover() {
	h := &status.hover
	status.cancelHoverDwell()
	item := status.items[h.iconID]
	if item == nil || !item.canHover() || h.opened {
		return
	}
	h.opened = true
	status.closeHoverPanel()
	if item.hoverPanel == nil {
		if h.hook == 0 {
			return // A hover menu needs a way to dismiss without activating its owner.
		}
		anchor := h.anchor
		h.menuID = item.id
		item.popMenu(status.window, anchor, false)
		h.menuID = 0
		// If the menu ended while the cursor was still over the icon, suppress
		// further shell movement reports until the pointer actually leaves.
		var cursor point
		if !item.removed.Load() {
			if ok, _, _ := getCursorPos.Call(uintptr(unsafe.Pointer(&cursor))); ok != 0 && hoverContains(anchor, cursor) {
				h.iconID, h.anchor, h.opened, h.postedLeave = item.id, anchor, true, false
			} else {
				status.leaveHover()
			}
		}
		status.syncHoverMouseHook()
		return
	}
	panel, ok := item.hoverPanel(h.anchor).(*nativeWindow)
	if !ok || panel == nil || panel.closed.Load() {
		return
	}
	if h.iconID != item.id || item.hoverPanel == nil {
		panel.Close()
		return
	}
	h.panelID, h.panel = item.id, panel
	panel.state.hoverStatus = status
}

func (status *statusState) leaveHover() {
	h := &status.hover
	status.cancelHoverDwell()
	h.iconID, h.opened, h.postedLeave = 0, false, false
	status.startHoverGap()
	status.syncHoverMouseHook()
}

func (status *statusState) startHoverGap() {
	h := &status.hover
	if h.panel != nil && h.gapUntil == 0 {
		h.gapUntil = monotonicNow() + statusHoverGap*int64(time.Millisecond)
		setTimer.Call(status.window, statusHoverGapTimer, statusHoverGap, 0)
	}
}

func (status *statusState) cancelHoverGap() {
	if status.hover.gapUntil != 0 {
		killTimer.Call(status.window, statusHoverGapTimer)
		status.hover.gapUntil = 0
	}
}

func (status *statusState) finishHoverGap() {
	h := &status.hover
	if h.gapUntil == 0 || monotonicNow() < h.gapUntil {
		return
	}
	status.cancelHoverGap()
	var cursor point
	var bounds rect
	if panel := h.panel; panel != nil && !panel.closed.Load() {
		if ok, _, _ := getCursorPos.Call(uintptr(unsafe.Pointer(&cursor))); ok != 0 {
			if h.iconID == h.panelID && hoverContains(h.anchor, cursor) {
				return
			}
			if ok, _, _ := getWindowRect.Call(panel.state.window, uintptr(unsafe.Pointer(&bounds))); ok != 0 &&
				cursor.X >= bounds.Left && cursor.X < bounds.Right && cursor.Y >= bounds.Top && cursor.Y < bounds.Bottom {
				return
			}
		}
	}
	status.closeHoverPanel()
}

func (status *statusState) closeHoverPanel() {
	h := &status.hover
	status.cancelHoverGap()
	if panel := h.panel; panel != nil {
		h.panelID, h.panel = 0, nil
		panel.state.hoverStatus = nil
		panel.Close()
	}
}

func (status *statusState) stopItemHover(id uint32) {
	if status.hover.panelID == id {
		status.closeHoverPanel()
	}
	if status.hover.iconID == id {
		status.leaveHover()
	}
	if status.hover.menuID == id {
		endStatusMenu.Call()
	}
}

func (status *statusState) resetHover() {
	status.closeHoverPanel()
	if status.hover.menuID != 0 {
		endStatusMenu.Call()
		status.hover.menuID = 0
	}
	status.leaveHover()
}

// The shell owns the icon's window. It reports NIN_POPUPCLOSE in the main
// tray, but can omit it in overflow. Watch only while a panel is requested
// or a native menu is tracking a hover. The panel tracks its own mouse leave.
func (status *statusState) syncHoverMouseHook() {
	h := &status.hover
	needed := status.window != 0 && (h.watchEntry || h.iconID != 0 || h.menuID != 0)
	if needed && h.hook == 0 {
		module, _, _ := getModule.Call(0)
		h.hook, _, _ = setWindowsHookEx.Call(14, statusMouseHookCallback, module, 0) // WH_MOUSE_LL
	} else if !needed && h.hook != 0 {
		unhookWindowsHookEx.Call(h.hook)
		h.hook = 0
	}
}

func statusMouseHook(code, kind, data uintptr) uintptr {
	if int32(code) >= 0 && data != 0 {
		if value, ok := statusStates.Load(statusWindowHandle.Load()); ok {
			status := value.(*statusState)
			cursor := *(*point)(pointer(data)) // MSLLHOOKSTRUCT starts with a POINT.
			h := &status.hover
			switch kind {
			case wmMouseMove:
				if h.watchEntry && h.iconID == 0 && !h.probePending && statusTrayAt(cursor) {
					posted, _, _ := postMessage.Call(status.window, wmStatusProbeIcon, 0, 0)
					h.probePending = posted != 0
				}
				if h.iconID != 0 && !h.postedLeave && !hoverContains(h.anchor, cursor) {
					posted, _, _ := postMessage.Call(status.window, wmStatusHoverLeave, uintptr(h.iconID), 0)
					h.postedLeave = posted != 0
				}
			case wmLeftDown, wmRightDown, wmMiddleDown, wmXButtonDown:
				if h.menuID != 0 {
					packed := uintptr(uint32(cursor.X)) | uintptr(uint32(cursor.Y))<<32
					postMessage.Call(status.window, wmStatusHoverDismissMenu, uintptr(h.menuID), packed)
				}
			}
		}
	}
	result, _, _ := callNextHookEx.Call(0, code, kind, data)
	return result
}

// Some Explorer taskbars do not send popup or motion callbacks until activation.
// Observe movement only while an item requests a panel. Query Explorer for the
// exact icon outside the input hook, and only over a notification-area surface.
func (status *statusState) updateEntryWatch() {
	status.hover.watchEntry = false
	for _, item := range status.items {
		if item.hoverPanel != nil && !item.removed.Load() {
			status.hover.watchEntry = true
			break
		}
	}
	status.syncHoverMouseHook()
	if status.hover.watchEntry {
		status.probeIcon()
	}
}

func statusTrayAt(p point) bool {
	hwnd, _, _ := windowFromPoint.Call(uintptr(uint32(p.X)) | uintptr(uint32(p.Y))<<32)
	if hwnd == 0 {
		return false
	}
	root, _, _ := statusAncestor.Call(hwnd, 2) // GA_ROOT
	var name [64]uint16
	if n, _, _ := statusWindowClassName.Call(root, uintptr(unsafe.Pointer(&name[0])), uintptr(len(name))); n == 0 {
		return false
	}
	switch syscall.UTF16ToString(name[:]) {
	case "Shell_TrayWnd", "Shell_SecondaryTrayWnd", "NotifyIconOverflowWindow", "TopLevelWindowForOverflowXamlIsland":
		return true
	}
	return false
}
func (status *statusState) probeIcon() {
	status.hover.probePending = false
	if !status.hover.watchEntry || status.hover.iconID != 0 || status.hover.menuID != 0 {
		return
	}
	var p point
	if ok, _, _ := getCursorPos.Call(uintptr(unsafe.Pointer(&p))); ok == 0 || !statusTrayAt(p) {
		return
	}
	if item := status.panelAt(p, func(item *statusItem) platform.ScreenRect { return item.anchor(status.window, 0) }); item != nil {
		status.beginHover(item, p, false)
	}
}

func (status *statusState) panelAt(p point, bounds func(*statusItem) platform.ScreenRect) *statusItem {
	for _, item := range status.items {
		if item.hoverPanel != nil && item.canHover() && hoverContains(bounds(item), p) {
			return item
		}
	}
	return nil
}

func (status *statusState) dismissHoverMenuAt(packed uintptr) {
	window, _, _ := windowFromPoint.Call(packed)
	if window != 0 {
		owner, _, _ := statusWindowThread.Call(status.window, 0)
		thread, _, _ := statusWindowThread.Call(window, 0)
		if owner != 0 && thread == owner {
			var name [8]uint16
			statusWindowClassName.Call(window, uintptr(unsafe.Pointer(&name[0])), uintptr(len(name)))
			if name == [8]uint16{'#', '3', '2', '7', '6', '8'} {
				return
			}
		}
	}
	endStatusMenu.Call()
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
