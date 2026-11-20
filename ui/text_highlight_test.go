package ui

import (
	"github.com/vivaui/viva/gfx"
	"math"
	"testing"
)

func TestHighlightUsesUTF8ByteOffsetsAndPreservesClip(t *testing.T) {
	var view View
	canvas := testCanvas()
	tint := gfx.RGB(220, 190, 80)
	value := "上个月的设计资料 DESIGN"
	view.Draw(canvas, gfx.LightTheme(), gfx.Rect{Width: 400, Height: 80}, func(f *Frame) {
		f.Column().Clip().Width(350).Children(func() {
			f.Text(value).FontSize(14).Highlight("设计", tint)
			f.Text("following label").FontSize(14)
		})
	})
	var label gfx.Text
	for _, text := range canvas.Texts() {
		if text.Value == value {
			label = text
			break
		}
	}
	spans := canvas.TextSelectionStyle(nil, value, label.Style, len("上个月的"), len("上个月的设计"))
	found := false
	for _, rect := range canvas.Rectangles() {
		if rect.Color == tint {
			found = true
			if math.Abs(float64(rect.Bounds.X-label.Bounds.X-spans[0].X)) > .01 || math.Abs(float64(rect.Bounds.Width-spans[0].Width)) > .01 {
				t.Fatalf("highlight is not on the matching Chinese text: %+v", rect.Bounds)
			}
		}
	}
	if !found {
		t.Fatal("missing highlight")
	}
	for _, text := range canvas.Texts() {
		if text.Value == "following label" && text.HasClip && text.ClipOf.Width < 300 {
			t.Fatal("highlight leaked a clip into the next label")
		}
	}
}

func TestHighlightCaseFoldPreservesTheOriginalText(t *testing.T) {
	var view View
	canvas := testCanvas()
	tint := gfx.RGB(220, 190, 80)
	view.Draw(canvas, gfx.LightTheme(), gfx.Rect{Width: 400, Height: 80}, func(f *Frame) { f.Text("Design DESIGN").Highlight("design", tint) })
	matches := 0
	for _, rect := range canvas.Rectangles() {
		if rect.Color == tint {
			matches++
		}
	}
	if matches != 2 {
		t.Fatalf("case-insensitive highlights = %d, want 2", matches)
	}
	if canvas.Texts()[0].Value != "Design DESIGN" {
		t.Fatal("highlight changed the original label")
	}
}
