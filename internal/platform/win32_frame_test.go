package platform

import "testing"

func TestAltGrIsRightAltWithAFakedLeftControl(t *testing.T) {
	if !IsAltGr(true, true, false) {
		t.Error("right Alt with left Control is AltGr")
	}
	if IsAltGr(true, true, true) {
		t.Error("with the right Control down too it is a real chord")
	}
	if IsAltGr(false, true, false) || IsAltGr(true, false, false) {
		t.Error("Control alone and right Alt alone are not AltGr")
	}
}

func TestPunctuationShortcutKeys(t *testing.T) {
	for _, vk := range []uintptr{0xbc /* , */, 0xbd /* - */, 0xbb /* = */, 0xbf /* / */, 0xdb, 0xdc, 0xde, 0xe2} {
		if !ShortcutPunctuationKey(vk) {
			t.Errorf("%#x is a punctuation key", vk)
		}
	}
	for _, vk := range []uintptr{'A', '0', 0x70, 0x20, 0xba - 1, 0xc1, 0xe3} {
		if ShortcutPunctuationKey(vk) {
			t.Errorf("%#x is not", vk)
		}
	}
}

func TestMaximizeButtonSitsSecondFromTheRight(t *testing.T) {
	const width, strip = 800, 32
	for _, test := range []struct {
		x, y float32
		want bool
	}{
		{width - 2*WindowControlWidth, 0, true},
		{width - WindowControlWidth - 1, strip - 1, true},
		{width - WindowControlWidth, 10, false}, // the close button
		{width - 2*WindowControlWidth - 1, 10, false},
		{width - 60, strip, false},
		{width - 60, -1, false},
	} {
		if got := MaximizeButtonAt(test.x, test.y, width, strip); got != test.want {
			t.Errorf("MaximizeButtonAt(%v, %v) = %v", test.x, test.y, got)
		}
	}
}

func TestMaximizedClientLeavesAPixelForAnAutoHidingBar(t *testing.T) {
	l, tp, r, b := MaximizedClient(-8, -8, 1928, 1088, 8, 8, [4]bool{})
	if l != 0 || tp != 0 || r != 1920 || b != 1080 {
		t.Fatalf("no bar: %d %d %d %d", l, tp, r, b)
	}
	l, tp, r, b = MaximizedClient(-8, -8, 1928, 1088, 8, 8, [4]bool{EdgeIndexBottom: true})
	if l != 0 || tp != 0 || r != 1920 || b != 1079 {
		t.Fatalf("bar at the bottom: %d %d %d %d", l, tp, r, b)
	}
	l, tp, r, b = MaximizedClient(-8, -8, 1928, 1088, 8, 8, [4]bool{EdgeIndexLeft: true, EdgeIndexTop: true, EdgeIndexRight: true})
	if l != 1 || tp != 1 || r != 1919 || b != 1080 {
		t.Fatalf("bars at three edges: %d %d %d %d", l, tp, r, b)
	}
}
