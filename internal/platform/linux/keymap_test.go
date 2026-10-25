package linux

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/vivaui/viva/internal/input"
	"github.com/vivaui/viva/text"
)

// A US layout, then a Russian one, then the AltGr levels of the US one: the shape the
// server reports for "setxkbmap us,ru" with a third level on the letters.
func twoLayoutKeymap() *xcbKeymap {
	const (
		cyrillicEF    = 0x6c6 // ф
		cyrillicEFCap = 0x6e6 // Ф
	)
	return &xcbKeymap{first: 38, last: 40, perKeycode: 8, keysyms: []uint32{
		// 38: a / ф, with ä on AltGr in the first layout and ö in the second
		'a', 'A', cyrillicEF, cyrillicEFCap, 0xe4, 0xc4, 0xf6, 0xd6,
		// 39: 7 / & and the keypad-less digit row
		'7', '&', '7', '&', '{', 0, '{', 0,
		// 40: a key of the second layout only
		0x6c9, 0x6e9, 0x6c9, 0x6e9, 0, 0, 0, 0,
	}}
}

func TestKeysymFollowsTheLayoutAndLevelTheStateNames(t *testing.T) {
	keymap := twoLayoutKeymap()
	group2 := uint16(1 << xcbGroupShift)
	for _, test := range []struct {
		name    string
		keycode uint8
		state   uint16
		want    uint32
	}{
		{"first layout", 38, 0, 'a'},
		{"shifted", 38, xcbModShift, 'A'},
		{"second layout", 38, group2, 0x6c6},
		{"second layout shifted", 38, group2 | xcbModShift, 0x6e6},
		{"caps lock capitalizes a Cyrillic letter", 38, group2 | xcbModLock, 0x6e6},
		{"caps lock and shift cancel", 38, group2 | xcbModLock | xcbModShift, 0x6c6},
		{"caps lock leaves a digit alone", 39, xcbModLock, '7'},
		{"AltGr selects the third level", 38, xcbMod5, 0xe4},
		{"AltGr and Shift the fourth", 38, xcbMod5 | xcbModShift, 0xc4},
		{"a level the key lacks falls back to the base", 39, xcbMod5 | xcbModShift, '{'},
		{"AltGr in the second layout uses its own levels", 38, group2 | xcbMod5, 0xf6},
		{"a key the first layout lacks", 40, 0, 0x6c9},
		{"an unknown key", 3, 0, 0},
	} {
		if got := keymap.keysym(test.keycode, test.state); got != test.want {
			t.Errorf("%s: keysym = %#x, want %#x", test.name, got, test.want)
		}
	}
}

func TestAGroupTheKeyDoesNotHaveFallsBackToTheFirst(t *testing.T) {
	keymap := &xcbKeymap{first: 10, last: 10, perKeycode: 2, keysyms: []uint32{'x', 'X'}}
	if got := keymap.keysym(10, 1<<xcbGroupShift); got != 'x' {
		t.Fatalf("keysym = %#x", got)
	}
	if got := keymap.keysym(10, 2<<xcbGroupShift); got != 'x' {
		t.Fatalf("a third layout reached %#x", got)
	}
}

func TestShortcutTextIsLatinInEveryLayout(t *testing.T) {
	keymap := twoLayoutKeymap()
	group2 := uint16(1 << xcbGroupShift)
	for _, test := range []struct {
		keycode uint8
		state   uint16
		want    string
	}{
		{38, 0, "a"},
		{38, group2, "a"}, // Ctrl+ф is Ctrl+A
		{38, group2 | xcbModShift, "A"},
		{38, xcbMod5, "a"}, // AltGr does not change what the key stands for
		{40, group2, "и"},  // a key with no Latin letter stands for what it types
	} {
		if got := keymap.shortcutText(test.keycode, test.state); got != test.want {
			t.Errorf("shortcutText(%d, %#x) = %q, want %q", test.keycode, test.state, got, test.want)
		}
	}
	// A layout of Cyrillic letters only, as a single-layout Russian machine has.
	russian := &xcbKeymap{first: 38, last: 38, perKeycode: 2, keysyms: []uint32{0x6c6, 0x6e6}}
	if got := russian.shortcutText(38, 0); got != "ф" {
		t.Fatalf("shortcutText on a Cyrillic-only layout = %q", got)
	}
}

func TestKeypadTypesDigitsWithNumLockAndNavigatesWithout(t *testing.T) {
	keymap := &xcbKeymap{first: 79, last: 79, perKeycode: 2, keysyms: []uint32{keysymKPHome, 0xffb7}} // KP_Home, KP_7
	if got := keymap.text(79, 0); got != "" {
		t.Fatalf("keypad 7 without Num Lock typed %q", got)
	}
	if got := keymap.text(79, xcbMod2); got != "7" {
		t.Fatalf("keypad 7 with Num Lock typed %q", got)
	}
	if got := keymap.text(79, xcbMod2|xcbModShift); got != "" {
		t.Fatalf("Shift with Num Lock typed %q, want the navigation key", got)
	}
	if got := keymap.key(79); got != input.KeyHome {
		t.Fatalf("key = %v", got)
	}
}

func TestModifierMapLocatesAltGrAndNumLock(t *testing.T) {
	keymap := &xcbKeymap{first: 8, last: 12, perKeycode: 1, keysyms: []uint32{
		keysymShiftL, keysymLevel3, keysymNumLock, keysymModeSwitch, 'a',
	}}
	// Two keycodes for each of the eight modifiers: AltGr on Mod5, Num Lock on Mod2
	// and Mode_switch on Mod3.
	keycodes := make([]uint8, 16)
	keycodes[7*2] = 9
	keycodes[4*2+1] = 10
	keycodes[5*2] = 11
	keymap.setModifierMap(keycodes, 2)
	if keymap.masks.level3 != xcbMod5 || keymap.masks.numLock != xcbMod2 || keymap.masks.modeSwitch != 1<<5 {
		t.Fatalf("masks = %+v", keymap.masks)
	}
	// The evdev rules put Mode_switch on the modifier AltGr is on: that is the third
	// level, not the second layout.
	keymap.masks = keyMasks{level3: xcbMod5, modeSwitch: xcbMod5}
	two := twoLayoutKeymap()
	two.masks = keymap.masks
	if got := two.keysym(38, xcbMod5); got != 0xe4 {
		t.Fatalf("AltGr on a modifier shared with Mode_switch chose %#x", got)
	}
	keymap.masks = keyMasks{}
	if keymap.level3Mask() != xcbMod5 || keymap.numLockMask() != xcbMod2 {
		t.Fatal("the conventional bits are not the default")
	}
}

func TestKeysymRuneCoversTheLegacyBlocksAndTheKeypad(t *testing.T) {
	for _, test := range []struct {
		keysym uint32
		want   rune
	}{
		{'a', 'a'}, {0xe9, 'é'}, {0x01000416, 'Ж'},
		{0x6c1, 'а'}, {0x6e1, 'А'}, // Cyrillic_a, Cyrillic_A
		{0x7e1, 'α'}, {0x1a1, 'Ą'}, {0xce0, 'א'}, {0xda1, 'ก'}, {0x20ac, '€'}, {0x13bc, 'Œ'},
		{0xffb5, '5'}, {0xffab, '+'}, {0xffae, '.'}, {keysymKPSpace, ' '},
	} {
		if got, ok := keysymRune(test.keysym); !ok || got != test.want {
			t.Errorf("keysymRune(%#x) = %q, %v; want %q", test.keysym, got, ok, test.want)
		}
	}
	for _, keysym := range []uint32{0, keysymEnter, keysymKPEnter, keysymKPHome, keysymF1, 0xfe51, 0x6ff0} {
		if got, ok := keysymRune(keysym); ok {
			t.Errorf("keysymRune(%#x) = %q, want none", keysym, got)
		}
	}
}

func TestLegacyKeysymTableIsSortedForBinarySearch(t *testing.T) {
	for index := 1; index < len(legacyKeysymRunes); index++ {
		if legacyKeysymRunes[index-1].keysym >= legacyKeysymRunes[index].keysym {
			t.Fatalf("entry %d (%#x) is out of order", index, legacyKeysymRunes[index].keysym)
		}
	}
}

func TestClickTimingFollowsTheDesktop(t *testing.T) {
	interval, distance := clickTiming(xsettings{}, 2)
	if interval != defaultDoubleClick || distance != defaultDoubleClickDistance {
		t.Fatalf("defaults = %v, %v", interval, distance)
	}
	settings := xsettings{DoubleClickTime: 400, hasDoubleClick: true, DoubleClickDistance: 10, hasDoubleClickDist: true}
	interval, distance = clickTiming(settings, 2)
	if interval != 400*time.Millisecond || distance != 5 {
		t.Fatalf("timing = %v, %v; the distance is device pixels and a press is logical", interval, distance)
	}
}

func TestCaretBlinkFollowsTheDesktop(t *testing.T) {
	if blinks, interval := caretBlink(xsettings{}); !blinks || interval != defaultBlinkInterval {
		t.Fatalf("default = %v, %v", blinks, interval)
	}
	if blinks, _ := caretBlink(xsettings{CursorBlink: 0, hasCursorBlink: true}); blinks {
		t.Fatal("a desktop that turned blinking off still blinks")
	}
	// GTK's blink time is the whole cycle; the caret spends half of it in each state.
	if _, interval := caretBlink(xsettings{CursorBlinkTime: 1200, hasCursorBlinkTime: true, hasCursorBlink: true, CursorBlink: 1}); interval != 600*time.Millisecond {
		t.Fatalf("interval = %v", interval)
	}
}

func TestParseXSettingsReadsTheTimingSettings(t *testing.T) {
	settings, err := parseXSettings(settingsBlob(binary.LittleEndian, map[string]int32{
		"Net/DoubleClickTime": 350, "Net/DoubleClickDistance": 6, "Gtk/CursorBlink": 0, "Gtk/CursorBlinkTime": 900,
	}, nil))
	if err != nil {
		t.Fatal(err)
	}
	if interval, _ := clickTiming(settings, 1); interval != 350*time.Millisecond {
		t.Errorf("double click interval = %v", interval)
	}
	if blinks, _ := caretBlink(settings); blinks {
		t.Error("Gtk/CursorBlink=0 did not stop the blinking")
	}
}

func TestSurroundingLeavesOutTheCompositionAndPutsTheCaretWhereItAppears(t *testing.T) {
	state := surroundingState{
		Value: "ab" + "ni hao" + "cd", Selection: text.Range{Start: 8, End: 8},
		Composition: text.Range{Start: 2, End: 8}, Composing: true,
	}
	window := surroundingOf(state)
	if window.Text != "abcd" || window.Cursor != 2 || window.Anchor != 2 {
		t.Fatalf("window = %+v", window)
	}
	// "Delete the two characters before the caret" is "ab", which sits before the preedit.
	if r, ok := window.fieldRange(-2, 2); !ok || r != (text.Range{Start: 0, End: 2}) {
		t.Fatalf("range = %+v, %v", r, ok)
	}
	// The two after it are "cd", which sit after the preedit in the field.
	if r, ok := window.fieldRange(0, 2); !ok || r != (text.Range{Start: 8, End: 10}) {
		t.Fatalf("range = %+v, %v", r, ok)
	}
	if _, ok := window.fieldRange(-1, 2); ok {
		t.Fatal("a request across the text being composed was honored")
	}
	if _, ok := window.fieldRange(-3, 1); ok {
		t.Fatal("a request before the window was honored")
	}
}

func TestSurroundingReportsTheSelectionAndBoundsTheWindow(t *testing.T) {
	window := surroundingOf(surroundingState{Value: "hello world", Selection: text.Range{Start: 6, End: 11}})
	if window.Text != "hello world" || window.Anchor != 6 || window.Cursor != 11 {
		t.Fatalf("window = %+v", window)
	}
	long := make([]rune, 5000)
	for index := range long {
		long[index] = 'x'
	}
	window = surroundingOf(surroundingState{Value: string(long), Selection: text.Range{Start: 2500, End: 2500}})
	if length := len([]rune(window.Text)); length != 2*surroundingLimit || window.Cursor != surroundingLimit {
		t.Fatalf("window of %d characters with the caret at %d", length, window.Cursor)
	}
	// A request inside the window maps back by the window's start.
	if r, ok := window.fieldRange(-1, 1); !ok || r != (text.Range{Start: 2499, End: 2500}) {
		t.Fatalf("range = %+v, %v", r, ok)
	}
}

func BenchmarkKeysymAndRune(b *testing.B) {
	keymap := twoLayoutKeymap()
	state := uint16(1<<xcbGroupShift) | xcbModLock
	b.ReportAllocs()
	var sink rune
	for range b.N {
		keysym := keymap.keysym(38, state)
		sink, _ = keysymRune(keysym)
	}
	_ = sink
}

func BenchmarkComposeIdleKey(b *testing.B) {
	var compose composer
	b.ReportAllocs()
	for range b.N {
		compose.feed('a')
	}
}
