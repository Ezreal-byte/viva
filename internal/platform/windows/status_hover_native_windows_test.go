package windows

import (
	"os"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"unsafe"

	"github.com/vivaui/viva/internal/platform"
)

func TestStatusHoverNativeMenu(t *testing.T) {
	if os.Getenv("VIVA_TEST_NOTIFICATION_DESKTOP") != "1" {
		t.Skip("requires a private Windows desktop")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	thread, _, _ := kernel.NewProc("GetCurrentThreadId").Call()
	previous, _, _ := user.NewProc("GetThreadDesktop").Call(thread)
	name, _ := syscall.UTF16PtrFromString("VivaStatusHover-" + strconv.Itoa(os.Getpid()))
	desktop, _, err := user.NewProc("CreateDesktopW").Call(uintptr(unsafe.Pointer(name)), 0, 0, 0, 0x01ff, 0)
	if desktop == 0 {
		t.Fatal(err)
	}
	defer user.NewProc("CloseDesktop").Call(desktop)
	if ok, _, err := user.NewProc("SetThreadDesktop").Call(desktop); ok == 0 {
		t.Fatal(err)
	}
	defer user.NewProc("SetThreadDesktop").Call(previous)

	err = RunApplication(func(a platform.Application) error {
		app := a.(*windowsApplication)
		owner, err := app.OpenWindow(platform.Config{Title: "Typing", Width: 200, Height: 100})
		if err != nil {
			return err
		}
		defer owner.Close()
		if err := app.ensureStatusWindow(); err != nil {
			return err
		}
		state := app.status
		item := &statusItem{id: 2, application: app, hoverMenu: true, menu: func() []platform.MenuItem {
			return []platform.MenuItem{{Title: "Conversation", OnSelect: func() { t.Error("outside click chose a menu item") }}}
		}}
		handle := owner.(*nativeWindow).state.window
		getFocus := user.NewProc("GetFocus")
		setFocus.Call(handle)
		if focus, _, _ := getFocus.Call(); focus != handle {
			t.Fatalf("initial focus = %x, want %x", focus, handle)
		}

		var sawMenu bool
		packedOutside := uintptr(uint32(1)) | uintptr(uint32(1))<<32
		var timer uintptr
		callback := syscall.NewCallback(func(_ uintptr, _ uint32, _ uintptr, _ uint32) uintptr {
			killTimer.Call(state.window, timer)
			var gui struct {
				size, flags uint32
				windows     [6]uintptr
				caret       rect
			}
			gui.size = uint32(unsafe.Sizeof(gui))
			getGUIThreadInfo := user.NewProc("GetGUIThreadInfo")
			ok, _, _ := getGUIThreadInfo.Call(thread, uintptr(unsafe.Pointer(&gui)))
			sawMenu = ok != 0 && gui.flags&4 != 0 // GUI_INMENUMODE
			if focus, _, _ := getFocus.Call(); focus != handle {
				t.Errorf("hover menu changed keyboard focus to %x", focus)
			}
			statusProc(state.window, wmStatusHoverDismissMenu, uintptr(item.id), packedOutside)
			return 0
		})
		timer, _, _ = setTimer.Call(state.window, 41, 100, callback)
		if timer == 0 {
			t.Fatal("could not schedule the outside-click fixture")
		}
		defer killTimer.Call(state.window, timer)
		state.hover.menuID = item.id
		item.popMenu(state.window, platform.ScreenRect{X: 100, Y: 400, Width: 20}, false)
		state.hover.menuID = 0
		if !sawMenu {
			t.Error("the outside click did not run during the native menu loop")
		}
		if focus, _, _ := getFocus.Call(); focus != handle {
			t.Errorf("menu dismissal changed keyboard focus to %x", focus)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
