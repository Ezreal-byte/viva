package windows

import (
	"testing"
	"unsafe"

	"github.com/vivaui/viva/internal/platform"
)

func TestMinimumTrackingPreservesNativeMaximizeAndUnsetAxis(t *testing.T) {
	state := windowState{config: platform.Config{WindowMode: platform.WindowCustomTitlebar, MinWidth: 300}, scale: 1.25}
	info := minMaxInfo{MaxSize: point{1920, 1080}, MaxPosition: point{-8, -8}, MinTrack: point{120, 32}, MaxTrack: point{4096, 4096}}
	before := info
	state.constrainTracking(&info)
	if info.MinTrack.X != 375 || info.MinTrack.Y != before.MinTrack.Y {
		t.Fatalf("minimum tracking size = %+v", info.MinTrack)
	}
	info.MinTrack = before.MinTrack
	if info != before {
		t.Fatalf("minimum changed maximize/snap geometry: %+v", info)
	}
	if size := unsafe.Sizeof(minMaxInfo{}); size != 40 {
		t.Fatalf("MINMAXINFO ABI size = %d, want 40", size)
	}
}

func BenchmarkMinimumTrackSize(b *testing.B) {
	desktop, _, _ := user.NewProc("GetDesktopWindow").Call()
	if desktop == 0 {
		b.Fatal("no desktop window")
	}
	monitor, _, _ := monitorFromWindow.Call(desktop, monitorNearest)
	if monitor == 0 {
		b.Fatal("desktop has no monitor")
	}
	for _, mode := range []struct {
		name string
		mode platform.WindowMode
	}{{"custom", platform.WindowCustomTitlebar}, {"titled", platform.WindowTitled}} {
		b.Run(mode.name, func(b *testing.B) {
			state := windowState{window: desktop, config: platform.Config{WindowMode: mode.mode, MinWidth: 960, MinHeight: 600}, scale: 1.25}
			b.ReportAllocs()
			for b.Loop() {
				_ = state.minimumTrackSize()
			}
		})
	}
}
