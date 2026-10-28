package linux

import (
	"testing"
	"unsafe"

	"github.com/vivaui/viva/internal/platform"
)

func TestResizingOnTheSameDisplayReusesMinimumHints(t *testing.T) {
	monitor := monitorGeometry{Width: 1920, Height: 1080, Primary: true}
	win := &window{
		tracker:     &scaleTracker{monitors: []monitorGeometry{monitor}},
		minimumSize: &minimumSizeCache{monitor: monitor, scale: 1, valid: true},
	}
	// No X connection is installed: a cache hit must not query or write the X
	// server, including as the client width and height change.
	for _, dimensions := range [][2]uint32{{1200, 800}, {960, 600}, {1400, 900}} {
		win.width, win.height = dimensions[0], dimensions[1]
		win.updateMinimumSize(platform.Config{MinWidth: 960, MinHeight: 600})
	}
	if allocations := testing.AllocsPerRun(100, func() {
		win.updateMinimumSize(platform.Config{MinWidth: 960, MinHeight: 600})
	}); allocations != 0 {
		t.Fatalf("unchanged display allocated %g times", allocations)
	}
	t.Logf("optional minimum cache: %d bytes; per-window pointer: %d bytes", unsafe.Sizeof(minimumSizeCache{}), unsafe.Sizeof(win.minimumSize))
}

func TestNoMinimumKeepsTheNativeWindowUnconfigured(t *testing.T) {
	win := &window{}
	win.updateMinimumSize(platform.Config{})
	if win.minimumSize != nil {
		t.Fatal("window without a minimum allocated a cache")
	}
}
