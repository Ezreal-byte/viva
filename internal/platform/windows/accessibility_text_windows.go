package windows

import (
	"sync/atomic"
	"unsafe"

	"github.com/vivaui/viva/internal/platform"
)

const (
	uiaIsReadOnlyAttribute = 40015

	textRangeEndpointStart = 0
	textRangeEndpointEnd   = 1

	// SupportedTextSelection_Single.
	textSelectionSingle = 1
)

var iidTextRangeProvider = guid{0x5347ad7b, 0xc355, 0x46f8, [8]byte{0xaf, 0xf5, 0x90, 0x90, 0x33, 0x58, 0x2f, 0x63}}

// uiaTextRange is one ITextRangeProvider: a span of an element's text in code
// points, which is how the application counts. UI Automation never sees an
// offset, only the text, the moves and the comparisons, so the unit of the
// count is ours to choose. The range reads the element's text each time it is
// asked, and clamps itself to it, so a range outlives an edit without pointing
// outside the text.
type uiaTextRange struct {
	Table      *uintptr // first, as in uiaInterface: this points at the table
	Owner      *uiaProvider
	Start, End int
	References atomic.Uint32
}

// 3 IUnknown functions and the 18 of ITextRangeProvider.
var uiaTextRangeTable [21]uintptr

func initializeTextRangeTable(unknown []uintptr) {
	copy(uiaTextRangeTable[:3], []uintptr{
		callbackFor(uiaTextRangeQueryInterface), callbackFor(uiaTextRangeAddReference), callbackFor(uiaTextRangeRelease),
	})
	methods := [...]uintptr{
		callbackFor(uiaRangeClone), callbackFor(uiaRangeCompare), callbackFor(uiaRangeCompareEndpoints),
		callbackFor(uiaRangeExpandToEnclosingUnit), callbackFor(uiaRangeFindAttribute), callbackFor(uiaRangeFindText),
		callbackFor(uiaRangeGetAttributeValue), callbackFor(uiaRangeGetBoundingRectangles), callbackFor(uiaRangeGetEnclosingElement),
		callbackFor(uiaRangeGetText), callbackFor(uiaRangeMove), callbackFor(uiaRangeMoveEndpointByUnit),
		callbackFor(uiaRangeMoveEndpointByRange), callbackFor(uiaRangeSelect), callbackFor(uiaRangeNotSupported),
		callbackFor(uiaRangeNotSupported), callbackFor(uiaRangeScrollIntoView), callbackFor(uiaRangeGetChildren),
	}
	copy(uiaTextRangeTable[3:], methods[:])
}

func newTextRange(owner *uiaProvider, start, end int) *uiaTextRange {
	made := &uiaTextRange{Table: &uiaTextRangeTable[0], Owner: owner, Start: start, End: end}
	made.References.Store(1)
	uiaRanges.Store(uintptr(unsafe.Pointer(made)), made)
	return made
}

func (made *uiaTextRange) address() uintptr { return uintptr(unsafe.Pointer(made)) }

// rangeAt is the range an interface pointer names, or nil for a pointer that is
// not one of ours.
func rangeAt(address uintptr) *uiaTextRange {
	if address == 0 {
		return nil
	}
	value, ok := uiaRanges.Load(address)
	if !ok {
		return nil
	}
	return value.(*uiaTextRange)
}

// state is the text the range reads and where it lies in it, clamped.
func (made *uiaTextRange) state() (node platform.AccessibilityNode, text []rune, start, end int, ok bool) {
	node, ok = made.Owner.snapshot()
	if !ok {
		return node, nil, 0, 0, false
	}
	text = []rune(node.Value)
	start, end = max(0, min(made.Start, len(text))), max(0, min(made.End, len(text)))
	if start > end {
		start, end = end, start
	}
	return node, text, start, end, true
}

func uiaTextRangeQueryInterface(this, requested, result uintptr) uintptr {
	if result == 0 {
		return windowsResult(resultPointer)
	}
	*(*uintptr)(pointer(result)) = 0
	made := rangeAt(this)
	if made == nil || requested == 0 {
		return windowsResult(resultNoInterface)
	}
	if id := *(*guid)(pointer(requested)); id != iidIUnknown && id != iidTextRangeProvider {
		return windowsResult(resultNoInterface)
	}
	made.References.Add(1)
	*(*uintptr)(pointer(result)) = this
	return 0
}

func uiaTextRangeAddReference(this uintptr) uintptr {
	made := rangeAt(this)
	if made == nil {
		return 0
	}
	return uintptr(made.References.Add(1))
}

func uiaTextRangeRelease(this uintptr) uintptr {
	made := rangeAt(this)
	if made == nil {
		return 0
	}
	count := made.References.Add(^uint32(0))
	if count == 0 {
		uiaRanges.Delete(this)
	}
	return uintptr(count)
}

// textUnit maps TextUnit of UI Automation to ours. Format and page have no
// meaning for a field of one style that is not paged.
func textUnit(unit uintptr) platform.AccessibilityTextUnit {
	switch int32(unit) {
	case 0:
		return platform.TextUnitCharacter
	case 2:
		return platform.TextUnitWord
	case 3:
		return platform.TextUnitLine
	case 4:
		return platform.TextUnitParagraph
	}
	return platform.TextUnitDocument
}

// textNode finds the element a text pattern call is for.
func textNode(this uintptr) (*uiaProvider, platform.AccessibilityNode, []rune, uintptr) {
	provider, node, status := patternNode(this)
	if status != 0 {
		return nil, node, nil, status
	}
	return provider, node, []rune(node.Value), 0
}

// caret is the selected span of a text node, a bare caret at the end of the
// text for a field that does not have the keyboard.
func caret(node platform.AccessibilityNode, text []rune) (start, end int) {
	if !node.Selection.Valid {
		return len(text), len(text)
	}
	start, end = int(node.Selection.Start), int(node.Selection.End)
	if start > end {
		start, end = end, start
	}
	return max(0, min(start, len(text))), max(0, min(end, len(text)))
}

func writeRange(result uintptr, made *uiaTextRange) uintptr {
	if result == 0 {
		return windowsResult(resultPointer)
	}
	*(*uintptr)(pointer(result)) = made.address()
	return 0
}

func uiaTextGetSelection(this, result uintptr) uintptr {
	if result == 0 {
		return windowsResult(resultPointer)
	}
	*(*uintptr)(pointer(result)) = 0
	provider, node, text, status := textNode(this)
	if status != 0 {
		return status
	}
	start, end := caret(node, text)
	array, ok := unknownArrayData([]uintptr{newTextRange(provider, start, end).address()})
	if !ok {
		return windowsResult(resultOutOfMemory)
	}
	*(*uintptr)(pointer(result)) = array
	return 0
}

func uiaTextGetVisibleRanges(this, result uintptr) uintptr {
	if result == 0 {
		return windowsResult(resultPointer)
	}
	*(*uintptr)(pointer(result)) = 0
	provider, _, text, status := textNode(this)
	if status != 0 {
		return status
	}
	array, ok := unknownArrayData([]uintptr{newTextRange(provider, 0, len(text)).address()})
	if !ok {
		return windowsResult(resultOutOfMemory)
	}
	*(*uintptr)(pointer(result)) = array
	return 0
}

func uiaTextRangeFromChild(this, child, result uintptr) uintptr {
	return uiaTextGetDocumentRange(this, result)
}

// uiaTextRangeFromPoint answers with the caret. A glyph at a screen point is not
// something the application's text layout can be asked in this adapter, and the
// caret is where a reader following the user goes next.
func uiaTextRangeFromPoint(this, point, result uintptr) uintptr {
	if result == 0 {
		return windowsResult(resultPointer)
	}
	*(*uintptr)(pointer(result)) = 0
	provider, node, text, status := textNode(this)
	if status != 0 {
		return status
	}
	_, end := caret(node, text)
	return writeRange(result, newTextRange(provider, end, end))
}

func uiaTextGetDocumentRange(this, result uintptr) uintptr {
	if result == 0 {
		return windowsResult(resultPointer)
	}
	*(*uintptr)(pointer(result)) = 0
	provider, _, text, status := textNode(this)
	if status != 0 {
		return status
	}
	return writeRange(result, newTextRange(provider, 0, len(text)))
}

func uiaTextGetSupportedSelection(this, result uintptr) uintptr {
	return writeInt(result, textSelectionSingle)
}

func uiaRangeClone(this, result uintptr) uintptr {
	if result == 0 {
		return windowsResult(resultPointer)
	}
	*(*uintptr)(pointer(result)) = 0
	made := rangeAt(this)
	if made == nil {
		return windowsResult(resultInvalidArgument)
	}
	return writeRange(result, newTextRange(made.Owner, made.Start, made.End))
}

func uiaRangeCompare(this, other, result uintptr) uintptr {
	made, target := rangeAt(this), rangeAt(other)
	if made == nil {
		return windowsResult(resultInvalidArgument)
	}
	same := target != nil && made.Owner == target.Owner
	if same {
		_, _, start, end, ok := made.state()
		_, _, targetStart, targetEnd, targetOK := target.state()
		same = ok && targetOK && start == targetStart && end == targetEnd
	}
	return writeBool(result, same)
}

func endpointOf(start, end int, endpoint uintptr) int {
	if int32(endpoint) == textRangeEndpointEnd {
		return end
	}
	return start
}

func uiaRangeCompareEndpoints(this, endpoint, other, otherEndpoint, result uintptr) uintptr {
	made, target := rangeAt(this), rangeAt(other)
	if made == nil || target == nil {
		return windowsResult(resultInvalidArgument)
	}
	_, _, start, end, ok := made.state()
	_, _, targetStart, targetEnd, targetOK := target.state()
	if !ok || !targetOK {
		return windowsResult(resultElementNotAvailable)
	}
	difference := endpointOf(start, end, endpoint) - endpointOf(targetStart, targetEnd, otherEndpoint)
	switch {
	case difference < 0:
		return writeInt(result, -1)
	case difference > 0:
		return writeInt(result, 1)
	}
	return writeInt(result, 0)
}

func uiaRangeExpandToEnclosingUnit(this, unit uintptr) uintptr {
	made := rangeAt(this)
	if made == nil {
		return windowsResult(resultInvalidArgument)
	}
	_, text, start, end, ok := made.state()
	if !ok {
		return windowsResult(resultElementNotAvailable)
	}
	made.Start, made.End = platform.TextExpand(text, start, end, textUnit(unit))
	return 0
}

// uiaRangeFindAttribute finds no run with an attribute: the text has one style.
func uiaRangeFindAttribute(this, attribute, value, backward, result uintptr) uintptr {
	if result == 0 {
		return windowsResult(resultPointer)
	}
	*(*uintptr)(pointer(result)) = 0
	return 0
}

func uiaRangeFindText(this, pattern, backward, ignoreCase, result uintptr) uintptr {
	if result == 0 {
		return windowsResult(resultPointer)
	}
	*(*uintptr)(pointer(result)) = 0
	made := rangeAt(this)
	if made == nil {
		return windowsResult(resultInvalidArgument)
	}
	_, text, start, end, ok := made.state()
	if !ok {
		return windowsResult(resultElementNotAvailable)
	}
	// The search covers the range only. BOOL is 32 bits wide; the upper half of
	// the register is not part of it.
	within := text[start:end]
	backwards := backward&0xffffffff != 0
	from := 0
	if backwards {
		from = len(within)
	}
	found, foundEnd, ok := platform.TextFind(within, utf16Pointer(pattern), from, backwards, ignoreCase&0xffffffff != 0)
	if !ok {
		return 0
	}
	return writeRange(result, newTextRange(made.Owner, start+found, start+foundEnd))
}

// uiaRangeGetAttributeValue says only that the field is read-only or not;
// anything else is the "not supported" value UI Automation defines.
func uiaRangeGetAttributeValue(this, attribute, result uintptr) uintptr {
	if result == 0 {
		return windowsResult(resultPointer)
	}
	output := (*variant)(pointer(result))
	*output = variant{Type: variantEmpty}
	made := rangeAt(this)
	if made == nil {
		return windowsResult(resultInvalidArgument)
	}
	node, ok := made.Owner.snapshot()
	if !ok {
		return windowsResult(resultElementNotAvailable)
	}
	if int32(attribute) == uiaIsReadOnlyAttribute {
		*output = booleanVariant(valueReadOnly(node))
		return 0
	}
	if uiaGetReservedNotSupportedValue.Find() == nil {
		var unsupported uintptr
		uiaGetReservedNotSupportedValue.Call(uintptr(unsafe.Pointer(&unsupported)))
		if unsupported != 0 {
			*output = variant{Type: variantUnknown, Value: uint64(unsupported)}
		}
	}
	return 0
}

// uiaRangeGetBoundingRectangles gives the field's own rectangle: where a glyph
// falls inside it is the text layout's to say, and the field is the best bound
// a reader following the range can be given.
func uiaRangeGetBoundingRectangles(this, result uintptr) uintptr {
	if result == 0 {
		return windowsResult(resultPointer)
	}
	*(*uintptr)(pointer(result)) = 0
	made := rangeAt(this)
	if made == nil {
		return windowsResult(resultInvalidArgument)
	}
	node, ok := made.Owner.snapshot()
	if !ok {
		return windowsResult(resultElementNotAvailable)
	}
	bounds := made.Owner.Tree.providerBounds(made.Owner, node)
	array, ok := doubleArrayData([]float64{bounds.X, bounds.Y, bounds.Width, bounds.Height})
	if !ok {
		return windowsResult(resultOutOfMemory)
	}
	*(*uintptr)(pointer(result)) = array
	return 0
}

func uiaRangeGetEnclosingElement(this, result uintptr) uintptr {
	if result == 0 {
		return windowsResult(resultPointer)
	}
	*(*uintptr)(pointer(result)) = 0
	made := rangeAt(this)
	if made == nil {
		return windowsResult(resultInvalidArgument)
	}
	made.Owner.addReference()
	*(*uintptr)(pointer(result)) = made.Owner.simplePointer()
	return 0
}

func uiaRangeGetText(this, maxLength, result uintptr) uintptr {
	made := rangeAt(this)
	if made == nil {
		return windowsResult(resultInvalidArgument)
	}
	_, text, start, end, ok := made.state()
	if !ok {
		return windowsResult(resultElementNotAvailable)
	}
	runes := text[start:end]
	if limit := int32(maxLength); limit >= 0 {
		units := 0
		for index, r := range runes {
			width := 1
			if r >= 0x10000 {
				width = 2
			}
			if units+width > int(limit) {
				runes = runes[:index]
				break
			}
			units += width
		}
	}
	return writeString(result, string(runes))
}

func uiaRangeMove(this, unit, count, result uintptr) uintptr {
	made := rangeAt(this)
	if made == nil {
		return windowsResult(resultInvalidArgument)
	}
	_, text, start, end, ok := made.state()
	if !ok {
		return windowsResult(resultElementNotAvailable)
	}
	var moved int
	made.Start, made.End, moved = platform.TextMoveSpan(text, start, end, textUnit(unit), int(int32(count)))
	return writeInt(result, int32(moved))
}

func uiaRangeMoveEndpointByUnit(this, endpoint, unit, count, result uintptr) uintptr {
	made := rangeAt(this)
	if made == nil {
		return windowsResult(resultInvalidArgument)
	}
	_, text, start, end, ok := made.state()
	if !ok {
		return windowsResult(resultElementNotAvailable)
	}
	var moved int
	made.Start, made.End, moved = platform.TextMoveEndpoint(text, start, end, int32(endpoint) == textRangeEndpointEnd, textUnit(unit), int(int32(count)))
	return writeInt(result, int32(moved))
}

func uiaRangeMoveEndpointByRange(this, endpoint, other, otherEndpoint uintptr) uintptr {
	made, target := rangeAt(this), rangeAt(other)
	if made == nil || target == nil || made.Owner != target.Owner {
		return windowsResult(resultInvalidArgument)
	}
	_, _, start, end, ok := made.state()
	_, _, targetStart, targetEnd, targetOK := target.state()
	if !ok || !targetOK {
		return windowsResult(resultElementNotAvailable)
	}
	position := endpointOf(targetStart, targetEnd, otherEndpoint)
	if int32(endpoint) == textRangeEndpointEnd {
		end = position
		start = min(start, end)
	} else {
		start = position
		end = max(end, start)
	}
	made.Start, made.End = start, end
	return 0
}

func uiaRangeSelect(this uintptr) uintptr {
	made := rangeAt(this)
	if made == nil {
		return windowsResult(resultInvalidArgument)
	}
	node, _, start, end, ok := made.state()
	if !ok {
		return windowsResult(resultElementNotAvailable)
	}
	return made.Owner.act(node, platform.AccessibilityRequest{Action: platform.AccessibilitySetSelection, Start: int32(start), End: int32(end)})
}

// uiaRangeNotSupported is AddToSelection and RemoveFromSelection: a field has
// one selection, which Select sets.
func uiaRangeNotSupported(this uintptr) uintptr { return windowsResult(resultNotSupported) }

func uiaRangeScrollIntoView(this, alignToTop uintptr) uintptr { return 0 }

func uiaRangeGetChildren(this, result uintptr) uintptr {
	if result == 0 {
		return windowsResult(resultPointer)
	}
	array, ok := unknownArrayData(nil)
	if !ok {
		return windowsResult(resultOutOfMemory)
	}
	*(*uintptr)(pointer(result)) = array
	return 0
}
