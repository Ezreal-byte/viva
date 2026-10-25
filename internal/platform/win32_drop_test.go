package platform

import "testing"

func TestDropEffectPrefersACopyAndRefusesWhereFilesAreNotTaken(t *testing.T) {
	for _, test := range []struct {
		accepts bool
		allowed uint32
		want    uint32
	}{
		{true, 1 | 2 | 4, 1},
		{true, 2 | 4, 4},
		{true, 2, 2},
		{true, 0, 0},
		{false, 1 | 2 | 4, 0},
	} {
		if got := DropEffect(test.accepts, test.allowed); got != test.want {
			t.Errorf("DropEffect(%v, %d) = %d, want %d", test.accepts, test.allowed, got, test.want)
		}
	}
}
