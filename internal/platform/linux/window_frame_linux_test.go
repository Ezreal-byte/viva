//go:build linux

package linux

import (
	"errors"
	"os"
	"testing"
	"unsafe"

	"github.com/vivaui/viva/internal/platform"
)

func openFramedWindow(t *testing.T, config platform.Config) (*linuxApplication, *nativeWindow) {
	t.Helper()
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no X display")
	}
	t.Setenv("VIVA_LINUX_FRAME", "force")
	application := &linuxApplication{}
	opened, err := application.OpenWindow(config)
	if errors.Is(err, platform.ErrNoGraphics) {
		t.Skipf("no Vulkan device: %v", err)
	}
	if err != nil {
		t.Skipf("no window: %v", err)
	}
	t.Cleanup(application.closeAll)
	return application, opened.(*nativeWindow)
}

// The application sees the content: its size, and where it is on the screen. The
// margin around it, which holds the shadow, is the window manager's and Viva's.
func TestFramedWindowReportsItsContentNotItsMargin(t *testing.T) {
	_, window := openFramedWindow(t, platform.Config{
		Title: "framed", Width: 300, Height: 200, WindowMode: platform.WindowCustomTitlebar, Actions: &platform.WindowActions{},
	})
	win := window.state.win
	if win.margin == 0 {
		t.Fatal("the window has no margin")
	}
	scale := win.displayScale()
	if want := deviceLength(300, scale) + 2*win.margin; win.width != want {
		t.Fatalf("X window is %d wide, want content plus both margins = %d", win.width, want)
	}
	size := win.logicalSize()
	if size.Width != 300 || size.Height != 200 {
		t.Fatalf("logical size = %vx%v, want the 300x200 asked for", size.Width, size.Height)
	}
	frame, ok := window.Frame()
	if !ok || frame.Width != 300 || frame.Height != 200 {
		t.Fatalf("Frame = %+v (%v), want the content's 300x200", frame, ok)
	}
	originX, originY := win.rootOrigin()
	if want := float32(originX+int32(win.margin)) / scale; frame.X != want {
		t.Fatalf("Frame.X = %v, want the content's %v (window at %d, margin %d)", frame.X, want, originX, win.margin)
	}
	if want := float32(originY+int32(win.margin)) / scale; frame.Y != want {
		t.Fatalf("Frame.Y = %v, want %v", frame.Y, want)
	}
}

// Pointer positions are the content's: a press at the content's corner is at 0,0, and
// one in the grip of the margin is outside it.
func TestFramedWindowPointerPositionsStartAtTheContent(t *testing.T) {
	_, window := openFramedWindow(t, platform.Config{
		Title: "framed", Width: 300, Height: 200, WindowMode: platform.WindowCustomTitlebar, Actions: &platform.WindowActions{},
	})
	win := window.state.win
	for _, test := range []struct {
		kind         uint8
		windowX      int16
		windowY      int16
		wantX, wantY int16
	}{
		{xcbButtonPress, int16(win.margin), int16(win.margin), 0, 0},
		{xcbMotionNotify, int16(win.margin) + 40, int16(win.margin) + 7, 40, 7},
		{xcbButtonRelease, int16(win.margin) - 3, int16(win.margin) - 3, -3, -3},
	} {
		event := xcbInputEvent{ResponseType: test.kind, EventX: test.windowX, EventY: test.windowY}
		win.shiftPointer(test.kind, unsafe.Pointer(&event))
		if event.EventX != test.wantX || event.EventY != test.wantY {
			t.Fatalf("kind %d: %d,%d, want %d,%d", test.kind, event.EventX, event.EventY, test.wantX, test.wantY)
		}
	}
	// A key event has no pointer position to move.
	key := xcbInputEvent{ResponseType: xcbKeyPress, EventX: 9, EventY: 9}
	win.shiftPointer(xcbKeyPress, unsafe.Pointer(&key))
	if key.EventX != 9 || key.EventY != 9 {
		t.Fatal("a key event was moved")
	}
}

// A window that is not drawn with a frame has no margin and reports the whole window.
func TestWindowWithoutAFrameHasNoMargin(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no X display")
	}
	t.Setenv("VIVA_LINUX_FRAME", "force")
	application := &linuxApplication{}
	opened, err := application.OpenWindow(platform.Config{Title: "titled", Width: 300, Height: 200, WindowMode: platform.WindowTitled})
	if err != nil {
		t.Skipf("no window: %v", err)
	}
	defer application.closeAll()
	if win := opened.(*nativeWindow).state.win; win.margin != 0 || win.frameWanted {
		t.Fatalf("a titled window has margin %d", win.margin)
	}
}

// A move to a display of another scale keeps the content's logical size and
// gives the margin a whole number of pixels at the new scale.
func TestFramedWindowKeepsItsContentWhenTheScaleChanges(t *testing.T) {
	_, window := openFramedWindow(t, platform.Config{
		Title: "framed", Width: 300, Height: 200, WindowMode: platform.WindowCustomTitlebar, Actions: &platform.WindowActions{},
	})
	state := window.state
	if err := state.applyScale(2); err != nil {
		t.Fatal(err)
	}
	win := state.win
	if win.margin != frameMargin(2) || state.renderer.frame.margin != frameMargin(2) {
		t.Fatalf("margin %d (renderer %d), want %d at 2x", win.margin, state.renderer.frame.margin, frameMargin(2))
	}
	if size := win.logicalSize(); size.Scale != 2 {
		t.Fatalf("scale %v", size.Scale)
	}
}
