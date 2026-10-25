package platform

import "testing"

func TestWheelPixelsScalesNotchesByTheUsersSetting(t *testing.T) {
	for _, test := range []struct {
		name    string
		delta   int
		units   uint32
		page    float32
		pixels  float32
		precise bool
	}{
		{"one notch, the default of three lines", 120, 3, 600, 48, false},
		{"one notch down", -120, 3, 600, -48, false},
		{"two notches", 240, 3, 600, 96, false},
		{"one line a notch", 120, 1, 600, 16, false},
		{"ten lines a notch", 120, 10, 600, 160, false},
		{"a page a notch", 120, WheelPageScroll, 600, 600, false},
		{"scrolling turned off", 120, 0, 600, 0, false},
		{"a touchpad nudge is exact and precise", 30, 3, 600, 12, true},
		{"a touchpad nudge back", -7, 3, 600, -2.8, true},
		{"a notch and a half is precise", 180, 3, 600, 72, true},
	} {
		pixels, precise := WheelPixels(test.delta, test.units, test.page)
		if diff := pixels - test.pixels; diff > 0.001 || diff < -0.001 || precise != test.precise {
			t.Errorf("%s: WheelPixels(%d, %d, %v) = %v, %v; want %v, %v", test.name, test.delta, test.units, test.page, pixels, precise, test.pixels, test.precise)
		}
	}
}
