package platform

import "testing"

func TestTitlebarStripDefaultsToTheSystemTitlebar(t *testing.T) {
	for _, test := range []struct {
		height float32
		want   float32
	}{
		{0, DefaultTitlebarHeight},
		{-4, DefaultTitlebarHeight},
		{28, 28},
		{36, 36},
	} {
		if got := (Config{TitlebarHeight: test.height}).TitlebarStrip(); got != test.want {
			t.Errorf("TitlebarHeight %v gives a strip of %v, want %v", test.height, got, test.want)
		}
	}
}
