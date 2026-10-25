package linux

import "testing"

func TestVisiblePositionKeepsAWindowThatCanBeReached(t *testing.T) {
	monitors := []monitorGeometry{
		{X: 0, Y: 0, Width: 1920, Height: 1080, Primary: true},
		{X: 1920, Y: 0, Width: 1280, Height: 1024},
	}
	for _, test := range []struct{ x, y int }{{100, 100}, {2000, 300}, {-400, 10}, {1800, 1000}} {
		if x, y, moved := visiblePosition(monitors, test.x, test.y, 800, 600); moved || x != test.x || y != test.y {
			t.Fatalf("(%d, %d) moved to (%d, %d)", test.x, test.y, x, y)
		}
	}
}

func TestVisiblePositionBringsBackAWindowFromAMissingMonitor(t *testing.T) {
	monitors := []monitorGeometry{{X: 0, Y: 0, Width: 1920, Height: 1080, Primary: true}}
	for _, test := range []struct{ x, y int }{{2500, 100}, {-2000, 0}, {100, 3000}, {1900, 1070}} {
		x, y, moved := visiblePosition(monitors, test.x, test.y, 800, 600)
		if !moved || x != 560 || y != 240 {
			t.Fatalf("(%d, %d) -> (%d, %d), moved %v; want the middle of the primary monitor", test.x, test.y, x, y, moved)
		}
	}
}

func TestVisiblePositionCentersOnThePrimaryMonitorNotTheFirst(t *testing.T) {
	monitors := []monitorGeometry{
		{X: 0, Y: 0, Width: 1000, Height: 800},
		{X: 1000, Y: 0, Width: 2000, Height: 1000, Primary: true},
	}
	if x, y, moved := visiblePosition(monitors, 9000, 9000, 400, 200); !moved || x != 1800 || y != 400 {
		t.Fatalf("got (%d, %d), moved %v", x, y, moved)
	}
}

func TestVisiblePositionLeavesAWindowAloneWithoutMonitors(t *testing.T) {
	if x, y, moved := visiblePosition(nil, 50, 60, 400, 200); moved || x != 50 || y != 60 {
		t.Fatalf("got (%d, %d), moved %v", x, y, moved)
	}
}

func TestUsableAreaTakesTheWorkAreaOnlyWhereItOverlaps(t *testing.T) {
	left := pixelRect{0, 0, 1920, 1080}
	right := pixelRect{1920, 0, 1280, 1024}
	// A panel across the bottom of the left monitor; the right monitor is whole.
	work := pixelRect{0, 0, 3200, 1040}
	if got := usableArea(left, work, true); got != (pixelRect{0, 0, 1920, 1040}) {
		t.Fatalf("left = %+v", got)
	}
	if got := usableArea(right, work, true); got != (pixelRect{1920, 0, 1280, 1024}) {
		t.Fatalf("right = %+v", got)
	}
	// A work area on the other monitor says nothing about this one.
	if got := usableArea(right, pixelRect{0, 0, 1920, 1000}, true); got != right {
		t.Fatalf("disjoint work area changed the monitor: %+v", got)
	}
	if got := usableArea(left, pixelRect{}, false); got != left {
		t.Fatalf("no work area: %+v", got)
	}
}
