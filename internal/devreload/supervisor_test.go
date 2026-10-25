package devreload

import (
	"testing"

	"github.com/vivaui/viva/internal/reloadwire"
	"github.com/vivaui/viva/ui"
)

func TestSupervisorPaintPreservesRealAndVisibleFocus(t *testing.T) {
	var canvas ui.Canvas
	var supervisor supervisor
	for _, testCase := range []struct {
		name         string
		focused      uint32
		focusVisible bool
	}{
		{name: "pointer focus", focused: 7},
		{name: "keyboard focus", focused: 7, focusVisible: true},
		{name: "pointer refocus", focused: 7},
		{name: "blur"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			canvas.SetInteraction(ui.Item(3), ui.Item(5), ui.Item(int(testCase.focused)), testCase.focusVisible)
			canvas.SetScroll(12.5, -8.25)
			supervisor.paint(&canvas, ui.Size{Width: 640, Height: 480})
			want := reloadwire.PaintRequest{
				Width: 640, Height: 480, Hovered: ui.Item(3), Pressed: ui.Item(5), Focused: ui.Item(int(testCase.focused)),
				FocusVisible: testCase.focusVisible, ScrollX: 12.5, ScrollY: -8.25,
			}
			if supervisor.lastRequest != want {
				t.Fatalf("paint request = %+v, want %+v", supervisor.lastRequest, want)
			}
		})
	}
}
