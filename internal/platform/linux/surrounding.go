package linux

import "github.com/vivaui/viva/text"

// What an input method is told of the text around the caret, and how what it asks
// to delete maps back to the field.
//
// A field keeps text being composed inside its value, so the text told to the
// engine leaves that part out and puts the caret where it appears: the engine
// has the composition itself. Only a window of surroundingLimit characters on
// each side of the caret is sent, so that a long document costs no more per
// keystroke than a short one.

const surroundingLimit = 512

type surrounding struct {
	// Text is the window; Cursor and Anchor are character offsets into it.
	Text           string
	Cursor, Anchor int

	// base is where Text begins in the field, in characters. The composition, if
	// any, was cut out at cutAt (an offset into Text) and was cutLength characters.
	base, cutAt, cutLength int
}

// surroundingOf builds the window from a field's state. The selection is ordered,
// so the anchor is its start and the caret its end.
func surroundingOf(state surroundingState) surrounding {
	runes := []rune(state.Value)
	selection, composition := state.Selection, state.Composition
	cutStart, cutLength := 0, 0
	if state.Composing && composition.End > composition.Start && composition.End <= len(runes) {
		cutStart, cutLength = composition.Start, composition.End-composition.Start
		runes = append(runes[:cutStart:cutStart], runes[composition.End:]...)
		selection = text.Range{Start: cutStart, End: cutStart}
	}
	selection.Start, selection.End = min(max(selection.Start, 0), len(runes)), min(max(selection.End, 0), len(runes))
	from, to := max(selection.Start-surroundingLimit, 0), min(selection.End+surroundingLimit, len(runes))
	result := surrounding{
		Text: string(runes[from:to]), Cursor: selection.End - from, Anchor: selection.Start - from,
		base: from, cutLength: cutLength,
	}
	if cutLength > 0 {
		result.cutAt = cutStart - from
	}
	return result
}

// surroundingState is the part of a field's state the window is made from.
type surroundingState struct {
	Value                  string
	Selection, Composition text.Range
	Composing              bool
}

// fieldRange maps a request to delete count characters starting offset from the
// caret onto the field's characters. It fails for a request outside the window,
// or one that reaches across text being composed.
func (window surrounding) fieldRange(offset, count int) (text.Range, bool) {
	length := len([]rune(window.Text))
	start := window.Cursor + offset
	end := start + count
	if count <= 0 || start < 0 || end > length {
		return text.Range{}, false
	}
	if window.cutLength > 0 && start < window.cutAt && end > window.cutAt {
		return text.Range{}, false
	}
	shift := func(position int, isEnd bool) int {
		if window.cutLength > 0 && (position > window.cutAt || position == window.cutAt && !isEnd) {
			position += window.cutLength
		}
		return window.base + position
	}
	return text.Range{Start: shift(start, false), End: shift(end, true)}, true
}
