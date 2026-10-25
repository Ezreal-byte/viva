package textmetrics

import (
	"unicode/utf8"

	"github.com/vivaui/viva/internal/uniprops"
)

// The complex scripts. The reference font has no glyphs for them, and a flat
// 0.6 em per character is the wrong width by a factor of two in either direction, so
// their advances come from tables made by tools/complexgen: the advances of the Noto Sans
// font of each script, shaped by HarfBuzz, in the same 2048 units per em as the Latin
// tables. Layout still never asks the platform, so the width of a word is the same on
// every machine; the platform draws it with its own font and shaper.
//
// The tables are not a shaper. Per script they hold the advance of each code point, and
// rules for what the shaper changes about a sum of advances:
//   - Arabic: a letter has an initial, medial, final or isolated advance, chosen by the
//     joining types of its neighbours (marks are transparent), and lam followed by alef
//     is one ligature;
//   - Devanagari, Bengali, Gujarati: a consonant followed by virama and consonant is the
//     half form of a conjunct, narrower than the full letter;
//   - Telugu, Kannada, Gurmukhi, Khmer, Myanmar: the consonant after the virama or coeng
//     is stacked under the one before and adds the advance of its subjoined form;
//   - marks add nothing, vowel signs the advance they add to a base letter; Thai, Lao,
//     Hebrew, Tamil, Malayalam, Armenian and Georgian are a sum of advances.
//
// The sum is checked against words shaped by HarfBuzz (testdata/shaped_samples.txt);
// ligatures the rules do not know, Bengali conjuncts above all, are the error.

type scriptKind uint8

const (
	scriptSimple   scriptKind = iota // a sum of per code point advances
	scriptArabic                     // joining letters
	scriptConjunct                   // the first consonant of a conjunct takes its half form
	scriptStacked                    // the second consonant of a conjunct stacks under the first
)

// noAdvance is a table entry for a code point the font lacks.
const noAdvance = 0xFFFF

type script struct {
	from, to rune
	kind     scriptKind
	mean     uint16      // the advance of a letter the table lacks
	adv      []uint16    // per code point from..to
	forms    [][3]uint16 // Arabic: initial, medial, final per code point from..to
	linker   rune        // virama or coeng
	consFrom rune        // the consonants that join, with join[c-consFrom]
	consTo   rune
	join     []uint16
	pairs    [][2]uint16 // conjuncts drawn as a glyph of their own: key, width (see conjunct)
}

// weightScale is by how much, in 1/1024, the lowercase letters of Inter's regular,
// medium, semibold and bold are wider than the regular ones: the other scripts' text
// is as much wider in a heavier weight.
var weightScale = [4]int{1024, 1037, 1050, 1063}

const (
	zwnj = 0x200C
	zwj  = 0x200D
)

// scriptOf is the table that covers r, or nil.
func scriptOf(r rune) *script {
	lo, hi := 0, len(scripts)
	for lo < hi {
		mid := (lo + hi) / 2
		s := &scripts[mid]
		switch {
		case r < s.from:
			hi = mid
		case r > s.to:
			lo = mid + 1
		default:
			return s
		}
	}
	return nil
}

// advanceOf is the table advance of r, a mark's being zero.
func (s *script) advanceOf(r rune) int {
	if a := s.adv[r-s.from]; a != noAdvance {
		return int(a)
	}
	if uniprops.Mark(r) {
		return 0
	}
	return int(s.mean)
}

// run is the width of the stretch of value that starts at i with a rune of s, and where
// it ends: every following rune of a complex script and the joiners between them.
func complexRun(value string, i int, weight int) (units, end int) {
	total := 0
	prevJoins := false // the previous letter connects to the next (Arabic)
	afterLinker := false
	skipAlef := false
	var s *script
	for i < len(value) {
		r, w := utf8.DecodeRuneInString(value[i:])
		if r == zwj || r == zwnj {
			if r == zwnj {
				prevJoins = false
			}
			i += w
			continue
		}
		if s == nil || r < s.from || r > s.to {
			if s = scriptOf(r); s == nil {
				break
			}
		}
		next := i + w
		switch s.kind {
		case scriptArabic:
			total += s.arabic(value, r, next, &prevJoins, &skipAlef)
		case scriptConjunct:
			total += s.conjunct(value, r, next)
		case scriptStacked:
			if r == s.linker && afterLinkerCandidate(s, value, i) {
				afterLinker = true
				break
			}
			if r >= s.consFrom && r <= s.consTo && afterLinker {
				if j := s.join[r-s.consFrom]; j != noAdvance {
					total += int(j)
					afterLinker = false
					break
				}
			}
			afterLinker = false
			total += s.advanceOf(r)
		default:
			total += s.advanceOf(r)
		}
		i = next
	}
	return total * weightScale[weight] >> 10, i
}

// afterLinkerCandidate reports whether the linker at value[i:] is followed by a consonant
// of s, so that the consonant is stacked.
func afterLinkerCandidate(s *script, value string, i int) bool {
	_, w := utf8.DecodeRuneInString(value[i:])
	if i+w >= len(value) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(value[i+w:])
	return r >= s.consFrom && r <= s.consTo
}

// conjunct is the advance of r in a script whose conjuncts shorten the first consonant:
// the half form when virama and another consonant follow (with nuktas and a zero width
// joiner allowed in between, a zero width non-joiner forbidding the conjunct).
func (s *script) conjunct(value string, r rune, next int) int {
	full := s.advanceOf(r)
	if r < s.consFrom || r > s.consTo {
		return full
	}
	half := s.join[r-s.consFrom]
	if half == noAdvance {
		return full
	}
	j := next
	for j < len(value) {
		m, w := utf8.DecodeRuneInString(value[j:])
		if m == s.linker || !uniprops.Mark(m) {
			break
		}
		j += w
	}
	if j >= len(value) {
		return full
	}
	m, w := utf8.DecodeRuneInString(value[j:])
	if m != s.linker {
		return full
	}
	j += w
	if j < len(value) {
		if m, w = utf8.DecodeRuneInString(value[j:]); m == zwnj {
			return full
		} else if m == zwj {
			return int(half)
		}
		if m >= s.consFrom && m <= s.consTo {
			return s.conjunctWidth(r, m, int(half))
		}
	}
	return full
}

// conjunctWidth is what the first consonant r of a conjunct with m adds to the width,
// given its half form: the whole width of a ligated conjunct less the full width of m,
// which is added when m itself is reached.
func (s *script) conjunctWidth(r, m rune, half int) int {
	key := uint16(r-s.consFrom)<<8 | uint16(m-s.consFrom)
	lo, hi := 0, len(s.pairs)
	for lo < hi {
		mid := (lo + hi) / 2
		switch p := s.pairs[mid]; {
		case p[0] < key:
			lo = mid + 1
		case p[0] > key:
			hi = mid
		default:
			return int(p[1]) - s.advanceOf(m)
		}
	}
	return half
}

// arabic is the advance of the Arabic character r (value[next:] follows it).
func (s *script) arabic(value string, r rune, next int, prevJoins, skipAlef *bool) int {
	join := uniprops.Joining(r)
	if join == uniprops.JoinT {
		return s.advanceOf(r) // a mark: transparent, the state stays
	}
	connectsBack := *prevJoins && (join == uniprops.JoinD || join == uniprops.JoinR || join == uniprops.JoinC)
	if *skipAlef { // the alef of a lam-alef ligature
		*skipAlef = false
		*prevJoins = false
		return 0
	}
	canConnectForward := join == uniprops.JoinD || join == uniprops.JoinL || join == uniprops.JoinC
	nextJoins := false
	var follower rune
	if canConnectForward {
		follower = nextBase(value, next)
		switch uniprops.Joining(follower) {
		case uniprops.JoinD, uniprops.JoinR, uniprops.JoinC:
			nextJoins = true
		}
	}
	*prevJoins = nextJoins
	if r == 0x644 && nextJoins {
		for k, alef := range [...]rune{0x622, 0x623, 0x625, 0x627} {
			if follower == alef {
				*skipAlef = true
				*prevJoins = false
				if connectsBack {
					return int(lamAlef[1][k])
				}
				return int(lamAlef[0][k])
			}
		}
	}
	isolated := s.advanceOf(r)
	if r < s.from || r > s.to || s.forms == nil {
		return isolated
	}
	forms := s.forms[r-s.from]
	pick := noAdvance
	switch {
	case connectsBack && nextJoins:
		pick = int(forms[1])
	case connectsBack:
		pick = int(forms[2])
	case nextJoins:
		pick = int(forms[0])
	}
	if pick == noAdvance || pick > 0xFFFE {
		return isolated
	}
	return pick
}

// nextBase is the first rune at or after value[i:] that is not transparent to joining
// (marks and joiners are looked through), 0 at the end.
func nextBase(value string, i int) rune {
	for i < len(value) {
		r, w := utf8.DecodeRuneInString(value[i:])
		if r == zwj {
			return r
		}
		if r == zwnj || uniprops.Joining(r) != uniprops.JoinT {
			return r
		}
		i += w
	}
	return 0
}

// hangul is the advance of the Hangul character r that follows a jamo of kind prev (0 for
// anything else), and its own kind: 1 leading consonant, 2 vowel (or an LV syllable),
// 3 trailing consonant (or an LVT syllable); ok is false for any other character. A
// syllable written as a sequence of conjoining jamo takes one em, like the precomposed
// syllable (UAX #29, GB6 to GB8), so a vowel or trailing consonant that continues one
// adds nothing.
func hangul(prev uint8, r rune) (advance int, kind uint8, ok bool) {
	switch {
	case r >= 0x1100 && r <= 0x115F, r >= 0xA960 && r <= 0xA97C:
		if prev == 1 {
			return 0, 1, true
		}
		return wideAdvance, 1, true
	case r >= 0x1160 && r <= 0x11A7, r >= 0xD7B0 && r <= 0xD7C6:
		if prev == 1 || prev == 2 {
			return 0, 2, true
		}
		return wideAdvance, 2, true
	case r >= 0x11A8 && r <= 0x11FF, r >= 0xD7CB && r <= 0xD7FB:
		if prev == 2 || prev == 3 {
			return 0, 3, true
		}
		return wideAdvance, 3, true
	case r >= 0xAC00 && r <= 0xD7A3:
		if (r-0xAC00)%28 == 0 {
			return wideAdvance, 2, true
		}
		return wideAdvance, 3, true
	}
	return 0, 0, false
}
