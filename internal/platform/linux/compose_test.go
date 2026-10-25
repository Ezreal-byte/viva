package linux

import "testing"

// type feeds a sequence of keysyms and returns what each one typed.
func typeKeys(compose *composer, keysyms ...uint32) []string {
	var typed []string
	for _, keysym := range keysyms {
		text, handled := compose.feed(keysym)
		switch {
		case !handled:
			character, _ := keysymRune(keysym)
			typed = append(typed, string(character))
		default:
			typed = append(typed, text)
		}
	}
	return typed
}

func TestDeadKeyAccentsTheNextLetter(t *testing.T) {
	var compose composer
	for _, test := range []struct {
		name string
		keys []uint32
		want []string
	}{
		{"acute e", []uint32{0xfe51, 'e'}, []string{"", "é"}},
		{"capital with diaeresis", []uint32{0xfe57, 'O'}, []string{"", "Ö"}},
		{"tilde n", []uint32{0xfe53, 'n'}, []string{"", "ñ"}},
		{"cedilla c", []uint32{0xfe5b, 'c'}, []string{"", "ç"}},
		{"a letter it cannot sit on keeps both", []uint32{0xfe51, 'x'}, []string{"", "´x"}},
		{"space types the ASCII character", []uint32{0xfe51, ' '}, []string{"", "'"}},
		{"diaeresis and space types a quotation mark", []uint32{0xfe57, ' '}, []string{"", "\""}},
		{"macron and space types the macron", []uint32{0xfe54, ' '}, []string{"", "¯"}},
		{"twice types the accent", []uint32{0xfe51, 0xfe51}, []string{"", "´"}},
		{"a key that is not a dead key is untouched", []uint32{'e'}, []string{"e"}},
	} {
		got := typeKeys(&compose, test.keys...)
		if len(got) != len(test.want) {
			t.Errorf("%s: typed %q, want %q", test.name, got, test.want)
			continue
		}
		for index := range got {
			if got[index] != test.want[index] {
				t.Errorf("%s: typed %q, want %q", test.name, got, test.want)
				break
			}
		}
		if compose.pending() {
			t.Errorf("%s: a sequence was left half typed", test.name)
			compose.reset()
		}
	}
}

func TestComposeKeyMakesSequences(t *testing.T) {
	var compose composer
	for _, test := range []struct {
		keys []uint32
		want string
	}{
		{[]uint32{keysymMultiKey, '\'', 'e'}, "é"},
		{[]uint32{keysymMultiKey, '"', 'u'}, "ü"},
		{[]uint32{keysymMultiKey, '^', 'o'}, "ô"},
		{[]uint32{keysymMultiKey, 0xfe51, 'a'}, "á"},
		{[]uint32{keysymMultiKey, 'o', 'c'}, "©"},
		{[]uint32{keysymMultiKey, '<', '<'}, "«"},
		{[]uint32{keysymMultiKey, 's', 's'}, "ß"},
		{[]uint32{keysymMultiKey, '/', 'o'}, "ø"},
		{[]uint32{keysymMultiKey, 'o', '/'}, "ø"},
		{[]uint32{keysymMultiKey, 'q', 'q'}, ""},
	} {
		got := typeKeys(&compose, test.keys...)
		if last := got[len(got)-1]; last != test.want {
			t.Errorf("%#x: typed %q, want %q", test.keys, got, test.want)
		}
		if compose.pending() {
			t.Errorf("%#x: a sequence was left half typed", test.keys)
			compose.reset()
		}
	}
}

func TestEscapeAndBackspaceCancelASequenceAndAreConsumed(t *testing.T) {
	var compose composer
	compose.feed(0xfe51)
	if text, handled := compose.feed(keysymEscape); !handled || text != "" || compose.pending() {
		t.Fatalf("Escape: %q, %v, pending %v", text, handled, compose.pending())
	}
	compose.feed(keysymMultiKey)
	if _, handled := compose.feed(keysymBackspace); !handled || compose.pending() {
		t.Fatalf("Backspace left the sequence going (handled %v)", handled)
	}
}

func TestAKeyThatIsNotTextEndsASequenceAndStillActs(t *testing.T) {
	var compose composer
	compose.feed(0xfe51)
	if _, handled := compose.feed(keysymLeft); handled || compose.pending() {
		t.Fatalf("an arrow key was swallowed (handled %v) or left the sequence pending", handled)
	}
}

func TestAnUnknownDeadKeyTypesTheNextLetterAlone(t *testing.T) {
	var compose composer
	got := typeKeys(&compose, 0xfe60, 'a') // dead_belowdot, which the tables do not have
	if got[0] != "" || got[1] != "a" {
		t.Fatalf("typed %q", got)
	}
}

func TestResetAndNilComposerAreSafe(t *testing.T) {
	var compose composer
	compose.feed(0xfe51)
	compose.reset()
	if compose.pending() {
		t.Fatal("reset left a sequence")
	}
	var none *composer
	none.reset()
	if none.pending() {
		t.Fatal("a nil composer is pending")
	}
}

func TestDeadAccentTablesAreAligned(t *testing.T) {
	for _, accent := range deadAccents {
		if len([]rune(accent.base)) != len([]rune(accent.result)) {
			t.Errorf("accent %#x: %d letters, %d results", accent.keysym, len([]rune(accent.base)), len([]rune(accent.result)))
		}
	}
}
