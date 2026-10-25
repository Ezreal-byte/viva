package components

import (
	"slices"
	"testing"

	"github.com/vivaui/viva/ui"
)

func TestAccordionReturnsContentAndPaintsStableDisclosureGlyphs(t *testing.T) {
	theme := ui.LightTheme()
	var canvas ui.Canvas
	accordion := BeginAccordion(&canvas, theme, ui.Rect{X: 8, Y: 10, Width: 300, Height: 240})
	openContent := accordion.Item(ui.Item(11), "Account", true, 48)
	closedContent := accordion.Item(ui.Item(12), "Security", false, 64)
	if openContent.Height != 48 || closedContent.Height != 0 {
		t.Fatalf("content bounds = %+v and %+v", openContent, closedContent)
	}
	if len(nonBlockHits(canvas.Hits())) != 2 || nonBlockHits(canvas.Hits())[0].ID != ui.Item(11) || nonBlockHits(canvas.Hits())[1].ID != ui.Item(12) {
		t.Fatalf("accordion hits = %+v", nonBlockHits(canvas.Hits()))
	}
	if openContent.Y+openContent.Height+16+2 != closedContent.Y-52 {
		t.Fatalf("second item does not follow first content: open=%+v closed=%+v", openContent, closedContent)
	}
	if nonBlockHits(canvas.Hits())[0].Role != ui.RoleButton || !nonBlockHits(canvas.Hits())[0].Expanded || nonBlockHits(canvas.Hits())[1].Expanded {
		t.Fatalf("accordion semantics = %+v", nonBlockHits(canvas.Hits()))
	}
}

func TestAccordionDisabledAndRepeatedPaintIsStableAndAllocationFree(t *testing.T) {
	theme := ui.DarkTheme()
	var canvas ui.Canvas
	paint := func() {
		canvas.Reset()
		accordion := BeginAccordion(&canvas, theme, ui.Rect{Width: 280, Height: 180}).Disabled()
		accordion.Item(ui.Item(1), "One", true, 32)
		accordion.Item(ui.Item(2), "Two", false, 32)
	}
	paint()
	rectangles := slices.Clone(canvas.Rectangles())
	texts := slices.Clone(canvas.Texts())
	paint()
	if len(nonBlockHits(canvas.Hits())) != 2 || !nonBlockHits(canvas.Hits())[0].Disabled || !nonBlockHits(canvas.Hits())[0].Focusable || !nonBlockHits(canvas.Hits())[0].FocusableDisabled {
		t.Fatalf("disabled accordion semantics = %+v", nonBlockHits(canvas.Hits()))
	}
	if !slices.Equal(rectangles, canvas.Rectangles()) || !slices.Equal(texts, canvas.Texts()) {
		t.Fatal("identical accordions produced different command streams")
	}
	if allocations := testing.AllocsPerRun(100, paint); allocations != 0 {
		t.Fatalf("accordion paint allocations = %v", allocations)
	}
}
