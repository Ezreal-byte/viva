package windows

import (
	"testing"

	"github.com/vivaui/viva/ui"
)

func TestSideButtonsAreBackAndForward(t *testing.T) {
	for _, test := range []struct {
		wParam uintptr
		want   ui.PointerButton
		known  bool
	}{
		{1 << 16, ui.PointerButtonBack, true},
		{2<<16 | mkLButton, ui.PointerButtonForward, true},
		{3 << 16, ui.PointerButtonNone, false},
		{0, ui.PointerButtonNone, false},
	} {
		if got, known := sideButton(test.wParam); got != test.want || known != test.known {
			t.Errorf("sideButton(%#x) = %v, %v; want %v, %v", test.wParam, got, known, test.want, test.known)
		}
	}
}
