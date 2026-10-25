package linux

import "unicode/utf8"

// latin1 is text as the Latin-1 bytes of an ICCCM STRING property: a rune the
// character set lacks becomes a question mark. A window manager that reads the
// UTF-8 name (_NET_WM_NAME) never looks at it.
func latin1(text string) []byte {
	out := make([]byte, 0, len(text))
	for _, character := range text {
		switch {
		case character == utf8.RuneError, character > 0xff:
			out = append(out, '?')
		default:
			out = append(out, byte(character))
		}
	}
	return out
}
