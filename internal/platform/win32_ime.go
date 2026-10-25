package platform

import (
	"unicode/utf16"

	"github.com/vivaui/viva/text"
)

// ReconvertContext is the text an input method reconverts (IMR_RECONVERTSTRING)
// or is fed as context (IMR_DOCUMENTFEED): the line of value around the
// selection as UTF-16 units, at most limit of them, and where the selection lies
// in it, in units. ok is false when there is no selection to reconvert or it
// does not fit in limit.
func ReconvertContext(value string, selection text.Range, limit int) (units []uint16, start, length int, ok bool) {
	runes := []rune(value)
	selection.Start = max(0, min(selection.Start, len(runes)))
	selection.End = max(selection.Start, min(selection.End, len(runes)))
	if selection.Start == selection.End {
		return nil, 0, 0, false
	}
	first, last := selection.Start, selection.End
	for first > 0 && runes[first-1] != '\n' {
		first--
	}
	for last < len(runes) && runes[last] != '\n' {
		last++
	}
	width := func(from, to int) int {
		count := 0
		for _, r := range runes[from:to] {
			count += utf16.RuneLen(r)
		}
		return count
	}
	selected := width(selection.Start, selection.End)
	if selected > limit {
		return nil, 0, 0, false
	}
	for width(first, last) > limit {
		before, after := width(first, selection.Start), width(selection.End, last)
		if before >= after && first < selection.Start {
			first++
		} else if last > selection.End {
			last--
		} else {
			first++
		}
	}
	units = utf16.Encode(runes[first:last])
	return units, width(first, selection.Start), selected, true
}

// IME attributes of one unit of a composition string (ATTR_*).
const (
	imeAttrInput              = 0
	imeAttrTargetConverted    = 1
	imeAttrConverted          = 2
	imeAttrTargetNotConverted = 3
)

// CompositionTarget is the clause of a composition the input method is
// converting right now, the one drawn with the thick underline, as a range of
// runes of value. attrs holds one attribute per UTF-16 unit of value
// (GCS_COMPATTR). ok is false when no clause is the target, as while typing
// before any conversion.
func CompositionTarget(attrs []byte, value string) (target text.Range, ok bool) {
	unit, found := 0, false
	for index, r := range []rune(value) {
		width := utf16.RuneLen(r)
		inTarget := false
		if unit < len(attrs) {
			inTarget = attrs[unit] == imeAttrTargetConverted || attrs[unit] == imeAttrTargetNotConverted
		}
		unit += width
		if !inTarget {
			if found {
				return target, true
			}
			continue
		}
		if !found {
			target.Start, found = index, true
		}
		target.End = index + 1
	}
	return target, found
}
