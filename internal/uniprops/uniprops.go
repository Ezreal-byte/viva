// Package uniprops holds the Unicode character properties Viva's text code needs:
// Line_Break (UAX #14), Bidi_Class, paired brackets and mirrored glyphs (UAX #9),
// Grapheme_Cluster_Break with Extended_Pictographic and Indic_Conjunct_Break (UAX #29)
// and the Arabic joining type. The tables are generated from the Unicode Character
// Database by tools/unigen so that every platform sees the same properties
// whatever the Go release or the operating system knows; they are two stage tables, a
// lookup is two array reads and allocates nothing.
package uniprops

//go:generate go run ../../tools/unigen -ucd $UCD -out tables.go

// Bracket is an entry of the paired bracket table.
type Bracket struct {
	Char, Pair rune
	Open       bool
}

const (
	flagPictographic = 1 << iota
	flagInCB1
	flagInCB2
	flagWide
	flagPi
	flagPf
	flagUnassigned
	flagMark
)

// InCB is the Indic_Conjunct_Break property.
type InCB uint8

const (
	InCBNone InCB = iota
	InCBConsonant
	InCBLinker
	InCBExtend
)

// tuple is the property tuple of r: line break class, bidi class, grapheme class, flags
// and joining type.
func tuple(r rune) *[5]uint8 {
	switch {
	case uint32(r) < 0x40000:
		return &tuples[tupleBlocks[int(tupleIndex[r>>7])<<7|int(r&127)]]
	case r >= 0xE0000 && r < 0xE1000:
		return &tuples[tagTuples[r-0xE0000]]
	}
	return &tuples[otherTuple]
}

// LineBreak is the Line_Break class of r.
func LineBreak(r rune) LineBreakClass { return LineBreakClass(tuple(r)[0]) }

// Bidi is the Bidi_Class of r.
func Bidi(r rune) BidiClass { return BidiClass(tuple(r)[1]) }

// Grapheme is the Grapheme_Cluster_Break class of r.
func Grapheme(r rune) GraphemeClass { return GraphemeClass(tuple(r)[2]) }

// Joining is the Joining_Type of r.
func Joining(r rune) JoiningType { return JoiningType(tuple(r)[4]) }

func flags(r rune) uint8 { return tuple(r)[3] }

// GraphemeProps returns the three properties grapheme segmentation needs from
// one tuple read, instead of one table lookup per property.
func GraphemeProps(r rune) (GraphemeClass, InCB, bool) {
	t := tuple(r)
	return GraphemeClass(t[2]), InCB(t[3] >> 1 & 3), t[3]&flagPictographic != 0
}

// ExtendedPictographic reports the Extended_Pictographic property of r.
func ExtendedPictographic(r rune) bool { return flags(r)&flagPictographic != 0 }

// ConjunctBreak is the Indic_Conjunct_Break property of r.
func ConjunctBreak(r rune) InCB { return InCB(flags(r) >> 1 & 3) }

// Wide reports whether r has East Asian Width F or W.
func Wide(r rune) bool { return flags(r)&flagWide != 0 }

// InitialQuote reports the general category Pi of r.
func InitialQuote(r rune) bool { return flags(r)&flagPi != 0 }

// FinalQuote reports the general category Pf of r.
func FinalQuote(r rune) bool { return flags(r)&flagPf != 0 }

// Unassigned reports the general category Cn of r.
func Unassigned(r rune) bool { return flags(r)&flagUnassigned != 0 }

// Mark reports whether r is a combining mark (general category Mn, Mc or Me).
func Mark(r rune) bool { return flags(r)&flagMark != 0 }

// LookupBracket returns the paired bracket entry of r, if it is one.
func LookupBracket(r rune) (Bracket, bool) {
	lo, hi := 0, len(brackets)
	for lo < hi {
		mid := (lo + hi) / 2
		switch {
		case brackets[mid].Char < r:
			lo = mid + 1
		case brackets[mid].Char > r:
			hi = mid
		default:
			return brackets[mid], true
		}
	}
	return Bracket{}, false
}

// Mirror returns the character whose glyph is the mirror image of r's, or r itself
// (Bidi_Mirroring_Glyph).
func Mirror(r rune) rune {
	lo, hi := 0, len(mirrors)
	for lo < hi {
		mid := (lo + hi) / 2
		switch {
		case mirrors[mid][0] < r:
			lo = mid + 1
		case mirrors[mid][0] > r:
			hi = mid
		default:
			return mirrors[mid][1]
		}
	}
	return r
}
