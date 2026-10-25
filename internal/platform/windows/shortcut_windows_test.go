//go:build windows

package windows

import "testing"

func TestShortcutTextNormalizesWindowsVirtualKeys(t *testing.T) {
	for key, want := range map[uintptr]string{'A': "a", 'Z': "z", '0': "0", '9': "9", 0x70: ""} {
		if got := shortcutText(key); got != want {
			t.Fatalf("shortcutText(%d) = %q, want %q", key, got, want)
		}
	}
}

func TestShortcutKeyTextKeepsLettersAndDigitsLayoutIndependent(t *testing.T) {
	for key, want := range map[uintptr]string{'A': "a", '9': "9", 0x70: "", 0x20: ""} {
		if got := shortcutKeyText(key, 0, false); got != want {
			t.Fatalf("shortcutKeyText(%#x) = %q, want %q", key, got, want)
		}
	}
}

// TestShortcutKeyTextNamesPunctuationKeys checks the OEM keys under a layout whose
// characters are known, the US one and the Chinese one that types the same.
func TestShortcutKeyTextNamesPunctuationKeys(t *testing.T) {
	layout, _, _ := getKeyboardLayout.Call(0)
	if language := layout & 0xffff; language != 0x0409 && language != 0x0804 {
		t.Skipf("keyboard layout %#x is not US-like", language)
	}
	for _, test := range []struct {
		vk, scan uintptr
		shift    bool
		want     string
	}{
		{0xbc, 0x33, false, ","}, {0xbc, 0x33, true, "<"},
		{0xbb, 0x0d, false, "="}, {0xbb, 0x0d, true, "+"},
		{0xbd, 0x0c, false, "-"}, {0xbf, 0x35, false, "/"},
	} {
		if got := shortcutKeyText(test.vk, test.scan<<16, test.shift); got != test.want {
			t.Errorf("shortcutKeyText(%#x, shift %v) = %q, want %q", test.vk, test.shift, got, test.want)
		}
	}
}
