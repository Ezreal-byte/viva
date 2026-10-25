package windows

import (
	"sync"
	"syscall"
	"unsafe"

	"github.com/vivaui/viva/internal/platform"
)

var setAppUserModelID = shell.NewProc("SetCurrentProcessExplicitAppUserModelID")
var appUserModelOnce sync.Once

// setApplicationIdentity names the process to the taskbar by the window class the
// application gave (platform.Config.Class), so its windows stay under one button
// and a pinned shortcut that carries the same name finds them. The first window
// decides; an application that gave no class keeps Windows' own grouping.
func setApplicationIdentity(class string) {
	appUserModelOnce.Do(func() {
		id := platform.AppUserModelID(class)
		if id == "" || setAppUserModelID.Find() != nil {
			return
		}
		if encoded, err := syscall.UTF16PtrFromString(id); err == nil {
			setAppUserModelID.Call(uintptr(unsafe.Pointer(encoded)))
		}
	})
}
