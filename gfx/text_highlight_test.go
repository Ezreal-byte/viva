package gfx

import (
	"math"
	"strings"
	"testing"
)

func TestLabelHighlightKeepsRTLAlignmentAndPaintsBeforeText(t *testing.T) {
	var canvas Canvas
	tint := RGB(220, 190, 80)
	canvas.LabelHighlighted("שלום design", Rect{Width: 350, Height: 24}, TextStyle{Size: 14}, AlignEnd, "DESIGN", tint)
	label := canvas.Texts()[0]
	start := strings.Index(label.Value, "design")
	spans := canvas.TextSelectionStyle(nil, label.Value, label.Style, start, start+len("design"))
	for _, rect := range canvas.Rectangles() {
		if rect.Color == tint {
			if rect.Order >= label.Order || math.Abs(float64(rect.Bounds.X-label.Bounds.X-spans[0].X)) > .01 {
				t.Fatalf("highlight lost text direction or paint order: %+v / %+v", rect, label)
			}
			return
		}
	}
	t.Fatal("missing RTL highlight")
}
