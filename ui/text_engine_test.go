package ui

import "testing"

// halfEngine measures every character as half the size and counts its calls.
type halfEngine struct{ widths, carets, selections, hits int }

func (e *halfEngine) Width(value string, size, weight float32) float32 {
	e.widths++
	return float32(len([]rune(value))) * size / 2
}

func (e *halfEngine) Caret(value string, size, weight float32, offset int) float32 {
	e.carets++
	return float32(len([]rune(value[:offset]))) * size / 2
}

func (e *halfEngine) Selection(dst []TextSpan, value string, size, weight float32, start, end int) []TextSpan {
	e.selections++
	return append(dst, TextSpan{X: float32(start) * size / 2, Width: float32(end-start) * size / 2})
}

func (e *halfEngine) OffsetAt(value string, size, weight, x float32) int {
	e.hits++
	return min(max(int(x/(size/2)+0.5), 0), len(value))
}

func TestCanvasAsksTheInstalledEngineForEveryTextQuery(t *testing.T) {
	var canvas Canvas
	engine := &halfEngine{}
	canvas.SetTextEngine(engine)

	if got := canvas.TextWidth("abcd", 10); got != 20 {
		t.Errorf("TextWidth = %v, want 20", got)
	}
	if got := canvas.TextCaret("abcd", 10, WeightRegular, 2); got != 10 {
		t.Errorf("TextCaret = %v, want 10", got)
	}
	if got := canvas.TextSelection(nil, "abcd", 10, WeightRegular, 1, 3); len(got) != 1 || got[0] != (TextSpan{X: 5, Width: 10}) {
		t.Errorf("TextSelection = %v", got)
	}
	if got := canvas.TextOffsetAt("abcd", 10, WeightRegular, 16); got != 3 {
		t.Errorf("TextOffsetAt = %v, want 3", got)
	}
	if *engine != (halfEngine{widths: 1, carets: 1, selections: 1, hits: 1}) {
		t.Errorf("calls = %+v, want one of each", *engine)
	}

	canvas.SetTextEngine(nil)
	if got := canvas.TextWidth("abcd", 10); got == 20 {
		t.Error("a canvas with no engine still measures with the removed one")
	}
}

func TestAnEmbeddedCanvasMeasuresWithTheHostsEngine(t *testing.T) {
	var host, child Canvas
	host.SetTextEngine(&halfEngine{})
	child.BeginEmbedded(&host, testMount(&host, 1))
	if got := child.TextWidth("abcd", 10); got != 20 {
		t.Errorf("embedded TextWidth = %v, want the host engine's 20", got)
	}
}

func TestAMeasurerOverridesTheEngineAndHitTestingFollowsIt(t *testing.T) {
	var canvas Canvas
	engine := &halfEngine{}
	canvas.SetTextEngine(engine)
	// One em per character: the click at 25 is nearest the boundary after 2 characters.
	canvas.SetMeasurer(func(value string, size, weight float32) float32 { return float32(len([]rune(value))) * size })
	if got := canvas.TextWidth("abcd", 10); got != 40 {
		t.Errorf("TextWidth = %v, want the measurer's 40", got)
	}
	if got := canvas.TextOffsetAt("abcd", 10, WeightRegular, 25); got != 2 && got != 3 {
		t.Errorf("TextOffsetAt = %v, want the boundary nearest 25", got)
	}
	if got := canvas.TextOffsetAt("日本語", 10, WeightRegular, 21); got != 6 {
		t.Errorf("TextOffsetAt over multibyte text = %v, want the byte offset 6 of the boundary nearest 21", got)
	}
	if engine.widths+engine.hits != 0 {
		t.Errorf("the engine was asked %+v although a measurer is installed", *engine)
	}
}

func TestTextQueriesTakeAnOffsetInsideACharacterToItsStart(t *testing.T) {
	var canvas Canvas
	canvas.SetTextEngine(&halfEngine{})
	// "é" is two bytes; offset 1 is inside it.
	if got, want := canvas.TextCaret("é", 10, WeightRegular, 1), canvas.TextCaret("é", 10, WeightRegular, 0); got != want {
		t.Errorf("caret inside a character = %v, at its start = %v", got, want)
	}
}

// The engine is an interface: a buffer handed straight to it would escape, and every
// paint of a field would allocate its selection spans.
func TestAnEngineSelectionDoesNotMakeTheCallersBufferEscape(t *testing.T) {
	var canvas Canvas
	canvas.SetTextEngine(&halfEngine{})
	canvas.TextSelection(nil, "abcd", 10, WeightRegular, 1, 3)
	if allocations := testing.AllocsPerRun(100, func() {
		var spans [4]TextSpan
		canvas.TextSelection(spans[:0], "abcd", 10, WeightRegular, 1, 3)
	}); allocations != 0 {
		t.Errorf("TextSelection allocated %v times per call", allocations)
	}
}
