package platform

import "unicode"

// AccessibilityTextUnit is a granularity at which assistive technology reads and
// moves through text. Offsets throughout this file are Unicode code points into
// the text, the units AccessibilitySelection uses; an adapter converts to its
// own (UTF-16 for UI Automation) at its boundary.
type AccessibilityTextUnit uint8

const (
	TextUnitCharacter AccessibilityTextUnit = iota
	// TextUnitWord is a run of letters and digits, or of punctuation, with the
	// spaces after it, as a screen reader speaks it.
	TextUnitWord
	// TextUnitLine is a row ended by a line break. A row the field wraps by
	// itself is not known here, so a wrapped paragraph is one line.
	TextUnitLine
	TextUnitParagraph
	TextUnitDocument
)

// TextUnitStart is the start of the unit that contains offset.
func TextUnitStart(text []rune, offset int, unit AccessibilityTextUnit) int {
	offset = clampOffset(text, offset)
	switch unit {
	case TextUnitDocument:
		return 0
	case TextUnitLine, TextUnitParagraph:
		for offset > 0 && text[offset-1] != '\n' {
			offset--
		}
		return offset
	case TextUnitWord:
		return wordStart(text, offset)
	default:
		return clusterStart(text, offset)
	}
}

// TextUnitEnd is the end of the unit that contains offset. At the end of the
// text it is the end of the text.
func TextUnitEnd(text []rune, offset int, unit AccessibilityTextUnit) int {
	offset = clampOffset(text, offset)
	if offset >= len(text) {
		return len(text)
	}
	switch unit {
	case TextUnitDocument:
		return len(text)
	case TextUnitLine, TextUnitParagraph:
		for offset < len(text) && text[offset] != '\n' {
			offset++
		}
		if offset < len(text) {
			offset++ // the break belongs to the line it ends
		}
		return offset
	case TextUnitWord:
		return wordEnd(text, wordStart(text, offset))
	default:
		return clusterEnd(text, clusterStart(text, offset))
	}
}

// TextNextBoundary is the boundary of unit that is count steps from offset,
// forward for a positive count and backward for a negative one, and how many
// steps it could take. A boundary is where a unit starts or ends.
func TextNextBoundary(text []rune, offset int, unit AccessibilityTextUnit, count int) (position, moved int) {
	position = clampOffset(text, offset)
	for count > 0 && position < len(text) {
		position = TextUnitEnd(text, position, unit)
		moved++
		count--
	}
	for count < 0 && position > 0 {
		if start := TextUnitStart(text, position, unit); start < position {
			position = start
		} else {
			position = TextUnitStart(text, position-1, unit)
		}
		moved--
		count++
	}
	return position, moved
}

// TextExpand is a span grown outward to whole units of the given size.
func TextExpand(text []rune, start, end int, unit AccessibilityTextUnit) (int, int) {
	start, end = orderSpan(text, start, end)
	newStart := TextUnitStart(text, start, unit)
	newEnd := TextUnitEnd(text, newStart, unit)
	if end > start {
		// A span that already ends on a boundary keeps its end.
		if last := TextUnitEnd(text, end-1, unit); last > newEnd {
			newEnd = last
		}
	}
	return newStart, newEnd
}

// TextMoveSpan moves a span by count units: it is made whole units first, then
// shifted so it covers the unit count steps away. It stops at the first or last
// unit of the text. moved is the number of steps taken.
func TextMoveSpan(text []rune, start, end int, unit AccessibilityTextUnit, count int) (newStart, newEnd, moved int) {
	start, end = TextExpand(text, start, end, unit)
	if count == 0 || len(text) == 0 {
		return start, end, 0
	}
	position := start
	for count > 0 {
		next := TextUnitEnd(text, position, unit)
		if next >= len(text) {
			break // position is already the last unit
		}
		position = next
		moved++
		count--
	}
	for count < 0 {
		if position == 0 {
			break
		}
		position = TextUnitStart(text, position-1, unit)
		moved--
		count++
	}
	return position, TextUnitEnd(text, position, unit), moved
}

// TextMoveEndpoint moves one end of a span (the start, or with moveEnd the end)
// by count unit boundaries. An end carried past the other drags it along, so a
// span is never inverted. moved is the number of boundaries crossed.
func TextMoveEndpoint(text []rune, start, end int, moveEnd bool, unit AccessibilityTextUnit, count int) (newStart, newEnd, moved int) {
	start, end = orderSpan(text, start, end)
	if moveEnd {
		end, moved = TextNextBoundary(text, end, unit, count)
		if end < start {
			start = end
		}
	} else {
		start, moved = TextNextBoundary(text, start, unit, count)
		if start > end {
			end = start
		}
	}
	return start, end, moved
}

// TextFind returns the first span of text equal to pattern at or after from, or
// before it when backward, and whether there was one.
func TextFind(text []rune, pattern string, from int, backward, ignoreCase bool) (start, end int, found bool) {
	needle := []rune(pattern)
	if len(needle) == 0 || len(needle) > len(text) {
		return 0, 0, false
	}
	equal := func(at int) bool {
		for index, want := range needle {
			got := text[at+index]
			if got == want {
				continue
			}
			if !ignoreCase || unicode.ToLower(got) != unicode.ToLower(want) {
				return false
			}
		}
		return true
	}
	last := len(text) - len(needle)
	if backward {
		for at := min(clampOffset(text, from)-len(needle), last); at >= 0; at-- {
			if equal(at) {
				return at, at + len(needle), true
			}
		}
		return 0, 0, false
	}
	for at := max(clampOffset(text, from), 0); at <= last; at++ {
		if equal(at) {
			return at, at + len(needle), true
		}
	}
	return 0, 0, false
}

func clampOffset(text []rune, offset int) int {
	return max(0, min(offset, len(text)))
}

func orderSpan(text []rune, start, end int) (int, int) {
	start, end = clampOffset(text, start), clampOffset(text, end)
	if start > end {
		start, end = end, start
	}
	return start, end
}

// zeroWidthJoiner ties the characters of an emoji sequence into one.
const zeroWidthJoiner = '\u200d'

func isCombining(r rune) bool {
	return unicode.In(r, unicode.Mn, unicode.Me, unicode.Mc) || r == zeroWidthJoiner
}

func isRegionalIndicator(r rune) bool { return r >= 0x1f1e6 && r <= 0x1f1ff }

// clusterStart walks back to the start of the character that holds offset: a
// base, the marks that follow it, and what a joiner ties to it.
func clusterStart(text []rune, offset int) int {
	if offset >= len(text) {
		return len(text)
	}
	for offset > 0 {
		previous := text[offset-1]
		if isCombining(text[offset]) || previous == zeroWidthJoiner {
			offset--
			continue
		}
		if isRegionalIndicator(text[offset]) && isRegionalIndicator(previous) {
			// A flag is two indicators; pair them from the run's start.
			run := offset
			for run > 0 && isRegionalIndicator(text[run-1]) {
				run--
			}
			if (offset-run)%2 == 1 {
				offset--
			}
		}
		break
	}
	return offset
}

func clusterEnd(text []rune, start int) int {
	if start >= len(text) {
		return len(text)
	}
	end := start + 1
	if isRegionalIndicator(text[start]) && end < len(text) && isRegionalIndicator(text[end]) {
		end++
	}
	for end < len(text) {
		switch {
		case isCombining(text[end]):
			end++
			if text[end-1] == zeroWidthJoiner && end < len(text) {
				end++ // the character a joiner ties on
			}
		default:
			return end
		}
	}
	return end
}

type wordClass uint8

const (
	classSpace wordClass = iota
	classBreak
	classWord
	classPunct
	classIdeograph
)

func classOf(r rune) wordClass {
	switch {
	case r == '\n' || r == '\r':
		return classBreak
	case unicode.IsSpace(r):
		return classSpace
	case unicode.Is(unicode.Han, r):
		return classIdeograph
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || unicode.IsMark(r):
		return classWord
	default:
		return classPunct
	}
}

// wordStart is the start of the word that holds offset: back over the spaces
// that trail a word, then back over the word. A break or an ideograph stands
// alone.
func wordStart(text []rune, offset int) int {
	if offset >= len(text) {
		return len(text)
	}
	position := offset
	for position > 0 && classOf(text[position]) == classSpace && classOf(text[position-1]) == classSpace {
		position--
	}
	if classOf(text[position]) == classSpace {
		// Spaces after a word belong to it; spaces after a break or at the
		// start of text are a word of their own.
		if position == 0 || classOf(text[position-1]) == classBreak {
			return position
		}
		position--
	}
	class := classOf(text[position])
	if class == classBreak || class == classIdeograph {
		return position
	}
	for position > 0 && classOf(text[position-1]) == class {
		position--
	}
	return position
}

// wordEnd is the end of the word that starts at start: the run of its class and
// the spaces after it.
func wordEnd(text []rune, start int) int {
	if start >= len(text) {
		return len(text)
	}
	class := classOf(text[start])
	end := start + 1
	switch class {
	case classBreak, classIdeograph:
		return end
	case classSpace:
	default:
		for end < len(text) && classOf(text[end]) == class {
			end++
		}
	}
	for end < len(text) && classOf(text[end]) == classSpace {
		end++
	}
	return end
}

// UTF16Offset is the offset in UTF-16 units of the code point at runeOffset in
// text; a negative or too large runeOffset gives the length of text in UTF-16
// units. Platform text APIs count in UTF-16, Viva's offsets in code points.
func UTF16Offset(text string, runeOffset int) int {
	units, index := 0, 0
	for _, r := range text {
		if runeOffset >= 0 && index >= runeOffset {
			break
		}
		units++
		if r >= 0x10000 {
			units++
		}
		index++
	}
	return units
}

// RuneOffset is the code point offset of the UTF-16 offset units in text. An
// offset inside a surrogate pair rounds down to the start of the character.
func RuneOffset(text string, units int) int {
	count, index := 0, 0
	for _, r := range text {
		width := 1
		if r >= 0x10000 {
			width = 2
		}
		if index+width > units {
			break
		}
		index += width
		count++
	}
	return count
}
