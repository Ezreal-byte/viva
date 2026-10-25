package bidi

import (
	"unicode/utf8"

	"github.com/vivaui/viva/internal/uniprops"
)

// BaseDirection is the paragraph direction rules P2 and P3 give text: RTL when its first
// strong character outside any isolate is R or AL, LTR (also for no strong character at
// all) otherwise. It allocates nothing and stops at the first strong character.
func BaseDirection(text string) Direction {
	depth := 0
	for i := 0; i < len(text); {
		c := text[i]
		if c < utf8.RuneSelf {
			if c|0x20 >= 'a' && c|0x20 <= 'z' && depth == 0 {
				return LTR
			}
			i++
			continue
		}
		r, w := utf8.DecodeRuneInString(text[i:])
		i += w
		switch class := uniprops.Bidi(r); {
		case isInitiator(class):
			depth++
		case class == cPDI:
			if depth > 0 {
				depth--
			}
		case depth > 0:
		case class == cL:
			return LTR
		case class == cR || class == cAL:
			return RTL
		}
	}
	return LTR
}
