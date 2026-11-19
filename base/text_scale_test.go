package base

import (
	"github.com/vivaui/viva/gfx"
	"testing"
)

func TestTextScaleAppliesToExplicitFontAndLineHeight(t *testing.T) {
	var view View
	canvas := testCanvas()
	view.Draw(canvas, gfx.LightTheme().WithTextScale(1.5), gfx.Rect{Width: 300, Height: 200}, func(frame *Frame) {
		frame.Column().Children(func() {
			frame.Text("scaled heading").FontSize(20).LineHeight(28)
		})
	})
	for _, label := range canvas.Texts() {
		if label.Value == "scaled heading" {
			if label.Style.Size != 30 {
				t.Fatalf("size=%v", label.Style.Size)
			}
			return
		}
	}
	t.Fatal("missing scaled label")
}
